package app

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/cockroachdb/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/infra/proxycapture"
	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func (m *grpcMocker) proxyServerStream(stream grpc.ServerStream, route *proxyroutes.Route, capture bool) error {
	req := dynamicpb.NewMessage(m.inputDesc)

	if err := stream.RecvMsg(req); err != nil {
		return err
	}

	return m.proxyServerStreamWithRequest(stream, route, req, capture)
}

//nolint:cyclop,funlen,nonamedreturns
func (m *grpcMocker) proxyServerStreamWithRequest(
	stream grpc.ServerStream,
	route *proxyroutes.Route,
	req *dynamicpb.Message,
	capture bool,
) (err error) {
	startTime := time.Now()

	desc := &grpc.StreamDesc{ServerStreams: true, ClientStreams: false}

	proxyCtx, cancel := route.WithStreamTimeout(proxyroutes.ForwardIncomingMetadata(stream.Context()), desc)
	defer cancel()

	historyEnabled := m.recorder != nil

	var (
		requestData      map[string]any
		historyResponses []map[string]any
		clientStream     grpc.ClientStream
	)

	if capture || historyEnabled {
		requestData = m.convertToMap(req)
	}

	// Single record point: covers setup failures, mid-stream errors on either
	// side, and clean completion alike (named return err).
	if historyEnabled {
		defer func() {
			m.recordProxyCall(stream.Context(), startTime, []map[string]any{requestData},
				historyResponses, responseHeadersFromClientStream(clientStream), err)
		}()
	}

	clientStream, err = route.Conn.NewStream(proxyCtx, desc, m.fullMethod)
	if err != nil {
		return err
	}

	if err = clientStream.SendMsg(req); err != nil {
		return err
	}

	if err = clientStream.CloseSend(); err != nil {
		return err
	}

	if header, headerErr := clientStream.Header(); headerErr == nil && len(header) > 0 {
		if setErr := stream.SetHeader(header); setErr != nil {
			return setErr
		}
	}

	responses := make([]any, 0, proxyMessagesInitCap)
	captureCtx := m.newCaptureRequestContext(stream.Context())
	recordDelay := route.Source.RecordDelay
	recorded := false

	lastMsgTime := startTime

	for {
		resp := dynamicpb.NewMessage(m.outputDesc)

		err = clientStream.RecvMsg(resp)

		if errors.Is(err, io.EOF) {
			err = nil

			break
		}

		if err != nil {
			if capture {
				m.recordCapturedStub(
					func() *stuber.Stub {
						return proxycapture.BuildServerStreamStub(
							m.fullServiceName, m.methodName, captureCtx.sessionID,
							requestData, captureCtx.headers, responses,
							captureMetadataFromClientStream(clientStream), err,
						)
					},
					recordDelay && !recorded,
					time.Since(startTime),
				)
			}

			return err
		}

		now := time.Now()

		// Convert once per message; the capture and history buffers share the
		// map except when capture mutates it with the delay marker.
		if capture || historyEnabled {
			var marked bool

			responses, historyResponses, marked = bufferProxyStreamMessage(
				responses, historyResponses, messageToAny(resp),
				capture, historyEnabled, recordDelay, now.Sub(lastMsgTime),
			)
			recorded = recorded || marked
		}

		lastMsgTime = now

		if err = stream.SendMsg(resp); err != nil {
			return err
		}
	}

	forwardUpstreamTrailer(stream, clientStream)

	if capture {
		m.recordCapturedStub(
			func() *stuber.Stub {
				return proxycapture.BuildServerStreamStub(
					m.fullServiceName, m.methodName, captureCtx.sessionID,
					requestData, captureCtx.headers, responses,
					captureMetadataFromClientStream(clientStream), nil,
				)
			},
			recordDelay && !recorded,
			time.Since(startTime),
		)
	}

	return nil
}

func (m *grpcMocker) proxyClientStream(stream grpc.ServerStream, route *proxyroutes.Route, capture bool) error {
	requestsToForward := make([]*dynamicpb.Message, 0, proxyMessagesInitCap)

	for {
		req := dynamicpb.NewMessage(m.inputDesc)

		err := stream.RecvMsg(req)

		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return err
		}

		requestsToForward = append(requestsToForward, req)
	}

	return m.proxyClientStreamWithRequests(stream, route, requestsToForward, capture)
}

//nolint:cyclop,funlen,nonamedreturns
func (m *grpcMocker) proxyClientStreamWithRequests(
	stream grpc.ServerStream,
	route *proxyroutes.Route,
	requestsToForward []*dynamicpb.Message,
	capture bool,
) (err error) {
	startTime := time.Now()

	desc := &grpc.StreamDesc{ServerStreams: false, ClientStreams: true}

	proxyCtx, cancel := route.WithStreamTimeout(proxyroutes.ForwardIncomingMetadata(stream.Context()), desc)
	defer cancel()

	historyEnabled := m.recorder != nil
	bookkeeping := capture || historyEnabled
	requests := make([]map[string]any, 0, proxyMessagesInitCap)

	var (
		historyResponses []map[string]any
		clientStream     grpc.ClientStream
	)

	// Single record point for every exit path (named return err).
	if historyEnabled {
		defer func() {
			m.recordProxyCall(stream.Context(), startTime, requests, historyResponses,
				responseHeadersFromClientStream(clientStream), err)
		}()
	}

	clientStream, err = route.Conn.NewStream(proxyCtx, desc, m.fullMethod)
	if err != nil {
		return err
	}

	captureCtx := m.newCaptureRequestContext(stream.Context())
	recordDelay := route.Source.RecordDelay

	for _, req := range requestsToForward {
		if bookkeeping {
			requests = append(requests, m.convertToMap(req))
		}

		if err = clientStream.SendMsg(req); err != nil {
			return err
		}
	}

	if err = clientStream.CloseSend(); err != nil {
		return err
	}

	if header, headerErr := clientStream.Header(); headerErr == nil && len(header) > 0 {
		if setErr := stream.SetHeader(header); setErr != nil {
			return setErr
		}
	}

	resp := dynamicpb.NewMessage(m.outputDesc)
	if err = clientStream.RecvMsg(resp); err != nil {
		if capture {
			m.recordCapturedStub(
				func() *stuber.Stub {
					return proxycapture.BuildClientStreamStub(
						m.fullServiceName, m.methodName, captureCtx.sessionID,
						requests, captureCtx.headers, nil,
						captureMetadataFromClientStream(clientStream), err,
					)
				},
				recordDelay, time.Since(startTime),
			)
		}

		return err
	}

	forwardUpstreamTrailer(stream, clientStream)

	var respEntry any
	if bookkeeping {
		respEntry = messageToAny(resp)

		if respMap, ok := respEntry.(map[string]any); ok && historyEnabled {
			historyResponses = append(historyResponses, respMap)
		}
	}

	if err = stream.SendMsg(resp); err != nil {
		return err
	}

	if capture {
		m.recordCapturedStub(
			func() *stuber.Stub {
				return proxycapture.BuildClientStreamStub(
					m.fullServiceName, m.methodName, captureCtx.sessionID,
					requests, captureCtx.headers, respEntry,
					captureMetadataFromClientStream(clientStream), nil,
				)
			},
			recordDelay, time.Since(startTime),
		)
	}

	return nil
}

func (m *grpcMocker) proxyBidiStream(stream grpc.ServerStream, route *proxyroutes.Route, capture bool) error {
	return m.proxyBidiStreamWithRequests(stream, route, nil, capture)
}

//nolint:funlen,nonamedreturns
func (m *grpcMocker) proxyBidiStreamWithRequests(
	stream grpc.ServerStream,
	route *proxyroutes.Route,
	prefetchedRequests []*dynamicpb.Message,
	capture bool,
) (err error) {
	startTime := time.Now()

	desc := &grpc.StreamDesc{ServerStreams: true, ClientStreams: true}

	proxyCtx, proxyCancel := route.WithStreamTimeout(proxyroutes.ForwardIncomingMetadata(stream.Context()), desc)
	defer proxyCancel()

	state := NewStreamCaptureState()
	state.startTime = startTime
	state.recordDelay = capture && route.Source.RecordDelay

	var clientStream grpc.ClientStream

	// Single record point for every exit path, including NewStream failure.
	if m.recorder != nil {
		defer func() {
			m.recordBidiProxyHistory(stream, clientStream, startTime, state, err)
		}()
	}

	clientStream, err = route.Conn.NewStream(proxyCtx, desc, m.fullMethod)
	if err != nil {
		return err
	}

	captureCtx := m.newCaptureRequestContext(stream.Context())

	errCh := make(chan error, proxyErrChanCap)

	bidiCtx, bidiCancel := context.WithCancel(proxyCtx)
	defer bidiCancel()

	go m.forwardBidiRequests(bidiCtx, stream, clientStream, prefetchedRequests, state, errCh)
	go m.forwardBidiResponses(bidiCtx, stream, clientStream, state, errCh)

	firstErr := <-errCh

	// Always apply a timeout so the select below does not hang
	// indefinitely when the second goroutine is blocked.
	timeout := route.Source.ReflectTimeout
	if timeout <= 0 {
		timeout = proxyBidiTimeoutFallback
	}

	var secondErr error
	select {
	case secondErr = <-errCh:
	case <-time.After(timeout):
		bidiCancel()
	}

	forwardUpstreamTrailer(stream, clientStream)

	if capture {
		requests, responses := state.Snapshot()
		needGlobalDelay := route.Source.RecordDelay && !state.HasTimedResponses()
		m.captureBidiResult(clientStream, captureCtx, requests, responses, firstErr, secondErr, needGlobalDelay, time.Since(startTime))
	}

	return selectCaptureError(firstErr, secondErr)
}

func trySendErr(ch chan<- error, err error) {
	select {
	case ch <- err:
	default:
	}
}

func (m *grpcMocker) forwardBidiRequests(
	bidiCtx context.Context,
	stream grpc.ServerStream,
	clientStream grpc.ClientStream,
	prefetchedRequests []*dynamicpb.Message,
	state *StreamCaptureState,
	errCh chan<- error,
) {
	defer func() {
		if r := recover(); r != nil {
			trySendErr(errCh, fmt.Errorf("forwardBidiRequests panic: %v", r)) //nolint:err113
		}
	}()

	for _, prefetched := range prefetchedRequests {
		if bidiCtx.Err() != nil {
			trySendErr(errCh, nil)

			return
		}

		state.AppendRequest(m.convertToMap(prefetched))

		if err := clientStream.SendMsg(prefetched); err != nil {
			trySendErr(errCh, err)

			return
		}
	}

	for {
		if bidiCtx.Err() != nil {
			trySendErr(errCh, nil)

			return
		}

		req := dynamicpb.NewMessage(m.inputDesc)

		err := stream.RecvMsg(req)

		if errors.Is(err, io.EOF) {
			closeSendErr := clientStream.CloseSend()
			trySendErr(errCh, closeSendErr)

			return
		}

		if err != nil {
			trySendErr(errCh, err)

			return
		}

		state.AppendRequest(m.convertToMap(req))

		if err = clientStream.SendMsg(req); err != nil {
			trySendErr(errCh, err)

			return
		}
	}
}

func (m *grpcMocker) forwardBidiResponses(
	bidiCtx context.Context,
	stream grpc.ServerStream,
	clientStream grpc.ClientStream,
	state *StreamCaptureState,
	errCh chan<- error,
) {
	defer func() {
		if r := recover(); r != nil {
			trySendErr(errCh, fmt.Errorf("forwardBidiResponses panic: %v", r)) //nolint:err113
		}
	}()

	for {
		if bidiCtx.Err() != nil {
			trySendErr(errCh, nil)

			return
		}

		resp := dynamicpb.NewMessage(m.outputDesc)

		err := clientStream.RecvMsg(resp)

		if errors.Is(err, io.EOF) {
			trySendErr(errCh, nil)

			return
		}

		if err != nil {
			trySendErr(errCh, err)

			return
		}

		now := time.Now()
		if respMap, ok := messageToAny(resp).(map[string]any); ok {
			state.AppendResponseWithTiming(respMap, now)
		}

		if err = stream.SendMsg(resp); err != nil {
			trySendErr(errCh, err)

			return
		}
	}
}

// forwardUpstreamTrailer copies the upstream client-stream trailer onto the
// downstream server stream (filtered), skipping the gRPC-web adapter which
// handles trailers itself.
func forwardUpstreamTrailer(stream grpc.ServerStream, clientStream grpc.ClientStream) {
	if trailer := clientStream.Trailer(); len(trailer) > 0 {
		if _, ok := stream.(*grpcwebAdapter); !ok {
			if t := ssmFilterMD(trailer); len(t) > 0 {
				stream.SetTrailer(t)
			}
		}
	}
}

// bufferProxyStreamMessage appends one converted upstream message to the
// capture and history buffers. The buffers share the map except when capture
// mutates it with the delay marker (history then keeps a clean copy). Returns
// marked=true when a delay marker was written.
func bufferProxyStreamMessage(
	responses []any,
	historyResponses []map[string]any,
	entry any,
	capture, historyEnabled, recordDelay bool,
	delay time.Duration,
) ([]any, []map[string]any, bool) {
	entryMap, isMap := entry.(map[string]any)
	marked := false

	if historyEnabled && isMap && len(historyResponses) < maxHistoryStreamMsgs {
		if capture && recordDelay {
			historyResponses = append(historyResponses, deepCopyMapAny(entryMap))
		} else {
			historyResponses = append(historyResponses, entryMap)
		}
	}

	if capture {
		if recordDelay && isMap {
			entryMap[stuber.GripMockKey] = map[string]any{"delay": delay.String()}
			marked = true
		}

		responses = append(responses, entry)
	}

	return responses, historyResponses, marked
}

// recordBidiProxyHistory writes a proxied bidi call into history with the
// capture-delay markers stripped from response copies. Snapshots the capture
// state itself so callers pay nothing when history is disabled.
func (m *grpcMocker) recordBidiProxyHistory(
	stream grpc.ServerStream,
	clientStream grpc.ClientStream,
	startTime time.Time,
	state *StreamCaptureState,
	callErr error,
) {
	if m.recorder == nil {
		return
	}

	requests, responses := state.Snapshot()

	historyResponses := make([]map[string]any, 0, min(len(responses), maxHistoryStreamMsgs))

	for _, r := range responses {
		if len(historyResponses) == maxHistoryStreamMsgs {
			break
		}

		clean := deepCopyMapAny(r)
		stuber.ExtractGripMockDelay(clean)
		historyResponses = append(historyResponses, clean)
	}

	m.recordProxyCall(stream.Context(), startTime, requests, historyResponses,
		responseHeadersFromClientStream(clientStream), callErr)
}
