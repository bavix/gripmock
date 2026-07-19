package deps

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/bavix/gripmock/v3/internal/app"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/httputil"
	infraTLS "github.com/bavix/gripmock/v3/internal/infra/tls"
)

const (
	maxBodyCapture      = 4 << 10 // 4 KiB
	grpcFrameHeaderSize = 5       // 1 flag byte + 4 byte length (gRPC/gRPC-Web)
)

const (
	gatewayReadHeaderTimeout = 10 * time.Second
	gatewayReadTimeout       = 30 * time.Second
	gatewayIdleTimeout       = 120 * time.Second
	gatewayMaxHeaderBytes    = 1 << 20
)

// GatewayServe starts the unified HTTP endpoint that handles both
// ConnectRPC and gRPC-web protocols on a single port.
func (b *Builder) GatewayServe(ctx context.Context) error {
	if b.config.Gateway.Port == "0" {
		return nil
	}

	gateway := b.newMultiProtocolGateway()

	router := mux.NewRouter()
	router.Handle("/{service}/{method}", gateway).Methods(http.MethodPost)

	srv := b.newGatewayServer(ctx, router)

	listener, err := b.listenGateway(ctx, srv)
	if err != nil {
		return err
	}

	b.ender.Add(srv.Shutdown)

	zerolog.Ctx(ctx).Info().
		Str("addr", listener.Addr().String()).
		Bool("tls", srv.TLSConfig != nil).
		Str("protocols", "connectrpc+grpc-web").
		Msg("Serving gateway (ConnectRPC + gRPC-Web)")

	return b.serveGateway(ctx, srv, listener)
}

func (b *Builder) newMultiProtocolGateway() *app.MultiProtocolGateway {
	var recorder history.Recorder
	if store := b.HistoryStore(); store != nil {
		recorder = store
	}

	g := app.NewMultiProtocolGateway(
		b.Budgerigar(),
		b.DescriptorRegistry(),
		recorder,
		b.ProxyRoutesRef(),
		b.StubValidator(),
		b.ErrorFormatter(),
	)

	return g
}

func (b *Builder) newGatewayServer(ctx context.Context, router *mux.Router) *http.Server {
	// Middleware order (innermost → outermost):
	//   router → access-log → gzip-decompress → compress → otel
	//
	// Must keep access-log INSIDE gzip/compress so it sees the
	// decompressed request body and the uncompressed response body.
	var handler http.Handler = router

	handler = gatewayAccessLogMiddleware(handler)
	handler = httputil.GzipRequestMiddleware(handler)
	handler = handlers.CompressHandler(handler)

	if b.config.OTel.Enabled {
		handler = otelhttp.NewHandler(handler, "gripmock-gateway")
	}

	return &http.Server{
		Addr:              b.config.Gateway.Addr,
		Handler:           handler,
		ReadHeaderTimeout: gatewayReadHeaderTimeout,
		ReadTimeout:       gatewayReadTimeout,
		IdleTimeout:       gatewayIdleTimeout,
		MaxHeaderBytes:    gatewayMaxHeaderBytes,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}
}

func (b *Builder) listenGateway(ctx context.Context, srv *http.Server) (net.Listener, error) {
	gatewayTLS := infraTLS.TLSConfig{
		CertFile:   b.config.GatewayTLS.CertFile,
		KeyFile:    b.config.GatewayTLS.KeyFile,
		ClientAuth: b.config.GatewayTLS.ClientAuth,
		CAFile:     b.config.GatewayTLS.CAFile,
		MinVersion: infraTLS.MinTLSVersion12,
	}

	if gatewayTLS.IsEnabled() {
		return b.tlsGatewayListener(srv, gatewayTLS)
	}

	setGatewayProtocols(srv)

	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", b.config.Gateway.Addr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to listen for gateway")
	}

	return listener, nil
}

func (b *Builder) tlsGatewayListener(srv *http.Server, gatewayTLS infraTLS.TLSConfig) (net.Listener, error) {
	tlsCfg, err := gatewayTLS.BuildTLSConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build gateway TLS config")
	}

	srv.TLSConfig = tlsCfg
	setGatewayProtocols(srv)

	tlsListener, err := tls.Listen("tcp", b.config.Gateway.Addr, srv.TLSConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gateway TLS listener")
	}

	return tlsListener, nil
}

func setGatewayProtocols(srv *http.Server) {
	srv.Protocols = func() *http.Protocols {
		var p http.Protocols
		p.SetHTTP1(true)

		if srv.TLSConfig != nil {
			p.SetHTTP2(true)
		} else {
			p.SetUnencryptedHTTP2(true)
		}

		return &p
	}()
}

// gatewayAccessLogMiddleware logs each gateway request on completion with
// fields consistent with the native gRPC access log format.
//
// The middleware reads the decompressed request body and captures the
// uncompressed response body. It must be placed INSIDE gzip/compress
// middleware (closer to the router) for this to work.
func gatewayAccessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		protocol := detectProtocol(r)
		reqBody := captureReqBody(r)
		rec := &captureResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rec, r)

		service, methodName := parseServiceMethod(r.URL.Path)
		meta := buildMetadata(r.Header)

		// Strip protocol frame headers for readable body logging.
		switch protocol {
		case "grpc-web":
			reqBody = stripGRPCFrame(reqBody)
			rec.bodyContent = stripGRPCFrame(rec.bodyContent)
		case "connectrpc":
			reqBody = stripConnectFrame(reqBody)
			rec.bodyContent = stripConnectFrame(rec.bodyContent)
		}

		log := zerolog.Ctx(r.Context()).Info().
			Str("gateway.code", http.StatusText(rec.statusCode)).
			Str("gateway.component", "server").
			Any("gateway.metadata", meta).
			Str("gateway.method", methodName).
			Str("gateway.service", service).
			Float64("gateway.time_ms", float64(time.Since(start).Microseconds())/1000.0). //nolint:mnd
			Str("peer.address", r.RemoteAddr)

		// Include request and response bodies (JSON-safe logging).
		log = logRequestBody(log, "gateway.request.content", r.Header, reqBody)
		log = logRequestBody(log, "gateway.response.content", rec.Header(), rec.bodyContent)

		log.Str("protocol", protocol).
			Msg("gateway call completed")
	})
}

// detectProtocol returns "grpc-web" or "connectrpc" based on Content-Type.
func detectProtocol(r *http.Request) string {
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/grpc-web") {
		return "grpc-web"
	}

	return "connectrpc"
}

// captureReqBody reads the request body (up to maxBodyCapture+1 bytes) and
// replaces r.Body so the next handler can still read it.
func captureReqBody(r *http.Request) string {
	if r.Body == nil {
		return ""
	}

	raw, _ := io.ReadAll(io.LimitReader(r.Body, maxBodyCapture+1))
	r.Body = io.NopCloser(bytes.NewReader(raw))

	if len(raw) > maxBodyCapture {
		return string(raw[:maxBodyCapture]) + "..."
	}

	return string(raw)
}

// buildMetadata returns a subset of request headers relevant for debugging.
func buildMetadata(h http.Header) map[string]any {
	m := make(map[string]any, 6) //nolint:mnd

	if v := h.Get("Content-Type"); v != "" {
		m["content-type"] = []string{v}
	}

	if v := h.Get("User-Agent"); v != "" {
		m["user-agent"] = []string{v}
	}

	if v := h.Get("Accept-Encoding"); v != "" {
		m["accept-encoding"] = []string{v}
	}

	if v := h.Get("Content-Encoding"); v != "" {
		m["content-encoding"] = []string{v}
	}

	if v := h.Get("X-Gripmock-Session"); v != "" {
		m["x-gripmock-session"] = []string{v}
	}

	if v := h.Get("Connect-Protocol-Version"); v != "" {
		m["connect-protocol-version"] = []string{v}
	}

	return m
}

// stripGRPCFrame removes the leading 5-byte gRPC frame header from a
// data frame (flag 0x00 or 0x01) and truncates the payload to the
// declared frame length. Any trailing frames (e.g. trailers) are cut
// off, leaving only the actual content (JSON/proto).
//
// Raw bodies (e.g. curl with application/json) start with 0x7B ('{')
// and are returned unchanged.
func stripGRPCFrame(data string) string {
	if len(data) < grpcFrameHeaderSize {
		return data
	}

	if data[0] != 0x00 && data[0] != 0x01 {
		return data
	}

	// declared = bytes 1-5 as big-endian uint32.
	declared := int(data[1])<<24 | int(data[2])<<16 | int(data[3])<<8 | int(data[4])

	payload := data[grpcFrameHeaderSize:]
	if declared < len(payload) {
		payload = payload[:declared]
	}

	return payload
}

// stripConnectFrame removes ConnectRPC envelope framing from log payloads.
// Connect envelopes are 5 bytes: 1 flag byte + 4 byte big-endian length.
// We skip the header and return only the payload up to the declared length.
// Raw bodies (JSON without framing) are returned unchanged.
func stripConnectFrame(data string) string {
	if len(data) < app.ConnectEnvelopeHeaderSize {
		return data
	}

	// Connect envelopes start with flag byte 0x00 (data) or 0x02 (end_stream).
	// JSON starts with 0x7B ('{') or 0x5B ('[').
	if data[0] != 0x00 && data[0] != 0x02 {
		return data
	}

	declared := int(data[1])<<24 | int(data[2])<<16 | int(data[3])<<8 | int(data[4])

	// Skip remaining envelope frames by returning the first payload.
	payload := data[app.ConnectEnvelopeHeaderSize:]
	if declared < len(payload) {
		payload = payload[:declared]
	}

	return payload
}

// isTextBody returns true when the content-type suggests a text-based
// payload (JSON). Binary protobuf bodies are skipped in the access log.
func isTextBody(h http.Header) bool {
	ct := h.Get("Content-Type")

	return strings.Contains(ct, "json")
}

// truncate shortens s to at most max bytes, appending "..." when cut.
func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}

	return s[:limit] + "..."
}

// parseServiceMethod splits /service/method from the URL path.
func parseServiceMethod(path string) (string, string) {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) >= 2 { //nolint:mnd
		return parts[0], parts[1]
	}

	if len(parts) == 1 {
		return parts[0], ""
	}

	return "", ""
}

// logRequestBody conditionally adds a body field to the zerolog event.
// If the content-type is text-based and the body is valid JSON, it is logged
// as RawJSON; otherwise as a truncated string. Returns the modified event.
func logRequestBody(log *zerolog.Event, key string, h http.Header, body string) *zerolog.Event {
	if !isTextBody(h) || body == "" {
		return log
	}

	if json.Valid([]byte(body)) {
		return log.RawJSON(key, []byte(body))
	}

	return log.Str(key, truncate(body, maxBodyCapture))
}

// captureResponseWriter wraps http.ResponseWriter to capture the HTTP
// status code and the response body (up to maxBodyCapture bytes).
type captureResponseWriter struct {
	http.ResponseWriter

	statusCode  int
	wroteHeader bool
	bodyContent string
}

func (w *captureResponseWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.statusCode = code
		w.wroteHeader = true
	}

	w.ResponseWriter.WriteHeader(code)
}

func (w *captureResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	// Accumulate all writes up to maxBodyCapture.
	remaining := maxBodyCapture - len(w.bodyContent)
	if remaining > 0 {
		if len(b) <= remaining {
			w.bodyContent += string(b)
		} else if remaining > 0 {
			w.bodyContent += string(b[:remaining])
		}
	}

	return w.ResponseWriter.Write(b)
}

func (b *Builder) serveGateway(ctx context.Context, srv *http.Server, listener net.Listener) error {
	ch := make(chan error, 1)

	go func() {
		defer close(ch)

		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			ch <- err
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-ch:
		return errors.Wrap(err, "gateway server failed")
	}
}
