package app

import (
	"io"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

//nolint:cyclop,funlen,wsl_v5
func (m *grpcMocker) proxyServerStreamWithRequest(
	stream grpc.ServerStream,
	route *proxyroutes.Route,
	req *dynamicpb.Message,
	capture bool,
) error {
	startTime := time.Now()
	proxyCtx, cancel := route.WithTimeout(proxyroutes.ForwardIncomingMetadata(stream.Context()))
	defer cancel()

	desc := &grpc.StreamDesc{ServerStreams: true, ClientStreams: false}
	clientStream, err := route.Conn.NewStream(proxyCtx, desc, m.fullMethod)
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

	responses := make([]map[string]any, 0, proxyMessagesInitCap)
	captureCtx := m.newCaptureRequestContext(stream.Context())
	requestData := convertToMap(req)
	recordDelay := route.Source.RecordDelay

	for {
		resp := dynamicpb.NewMessage(m.outputDesc)
		err = clientStream.RecvMsg(resp)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			if capture {
				m.recordCapturedStub(
					func() *stuber.Stub {
						return proxycapture.BuildServerStreamStub(
							m.fullServiceName, m.methodName, captureCtx.sessionID,
							requestData, captureCtx.headers, responses,
							responseHeadersFromClientStream(clientStream), err,
						)
					},
					recordDelay, time.Since(startTime),
				)
			}

			return err
		}

		responses = append(responses, messageToMap(resp))

		if err = stream.SendMsg(resp); err != nil {
			return err
		}
	}

	if trailer := clientStream.Trailer(); len(trailer) > 0 {
		stream.SetTrailer(trailer)
	}

	if capture {
		m.recordCapturedStub(
			func() *stuber.Stub {
				return proxycapture.BuildServerStreamStub(
					m.fullServiceName, m.methodName, captureCtx.sessionID,
					requestData, captureCtx.headers, responses,
					responseHeadersFromClientStream(clientStream), nil,
				)
			},
			recordDelay, time.Since(startTime),
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

//nolint:cyclop,funlen
func (m *grpcMocker) proxyClientStreamWithRequests(
	stream grpc.ServerStream,
	route *proxyroutes.Route,
	requestsToForward []*dynamicpb.Message,
	capture bool,
) error {
	startTime := time.Now()

	proxyCtx, cancel := route.WithTimeout(proxyroutes.ForwardIncomingMetadata(stream.Context()))
	defer cancel()

	desc := &grpc.StreamDesc{ServerStreams: false, ClientStreams: true}

	clientStream, err := route.Conn.NewStream(proxyCtx, desc, m.fullMethod)
	if err != nil {
		return err
	}

	requests := make([]map[string]any, 0, proxyMessagesInitCap)
	captureCtx := m.newCaptureRequestContext(stream.Context())
	recordDelay := route.Source.RecordDelay

	for _, req := range requestsToForward {
		requests = append(requests, convertToMap(req))

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
						responseHeadersFromClientStream(clientStream), err,
					)
				},
				recordDelay, time.Since(startTime),
			)
		}

		return err
	}

	if trailer := clientStream.Trailer(); len(trailer) > 0 {
		stream.SetTrailer(trailer)
	}

	if err = stream.SendMsg(resp); err != nil {
		return err
	}

	if capture {
		m.recordCapturedStub(
			func() *stuber.Stub {
				return proxycapture.BuildClientStreamStub(
					m.fullServiceName, m.methodName, captureCtx.sessionID,
					requests, captureCtx.headers, messageToMap(resp),
					responseHeadersFromClientStream(clientStream), nil,
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

func (m *grpcMocker) proxyBidiStreamWithRequests(
	stream grpc.ServerStream,
	route *proxyroutes.Route,
	prefetchedRequests []*dynamicpb.Message,
	capture bool,
) error {
	startTime := time.Now()

	proxyCtx, cancel := route.WithTimeout(proxyroutes.ForwardIncomingMetadata(stream.Context()))
	defer cancel()

	desc := &grpc.StreamDesc{ServerStreams: true, ClientStreams: true}

	clientStream, err := route.Conn.NewStream(proxyCtx, desc, m.fullMethod)
	if err != nil {
		return err
	}

	state := newBidiCaptureState()

	captureCtx := m.newCaptureRequestContext(stream.Context())

	errCh := make(chan error, proxyErrChanCap)

	go m.forwardBidiRequests(stream, clientStream, prefetchedRequests, state, errCh)

	go m.forwardBidiResponses(stream, clientStream, state, errCh)

	firstErr := <-errCh
	secondErr := <-errCh

	if trailer := clientStream.Trailer(); len(trailer) > 0 {
		stream.SetTrailer(trailer)
	}

	if capture {
		requests, responses := state.snapshot()

		m.captureBidiResult(clientStream, captureCtx, requests, responses, firstErr, secondErr, route.Source.RecordDelay, time.Since(startTime))
	}

	if firstErr != nil {
		return firstErr
	}

	if secondErr != nil {
		return secondErr
	}

	return nil
}

func (m *grpcMocker) forwardBidiRequests(
	stream grpc.ServerStream,
	clientStream grpc.ClientStream,
	prefetchedRequests []*dynamicpb.Message,
	state *bidiCaptureState,
	errCh chan<- error,
) {
	for _, prefetched := range prefetchedRequests {
		state.appendRequest(convertToMap(prefetched))

		if err := clientStream.SendMsg(prefetched); err != nil {
			errCh <- err

			return
		}
	}

	for {
		req := dynamicpb.NewMessage(m.inputDesc)

		err := stream.RecvMsg(req)
		if errors.Is(err, io.EOF) {
			errCh <- clientStream.CloseSend()

			return
		}

		if err != nil {
			errCh <- err

			return
		}

		state.appendRequest(convertToMap(req))

		if err = clientStream.SendMsg(req); err != nil {
			errCh <- err

			return
		}
	}
}

func (m *grpcMocker) forwardBidiResponses(
	stream grpc.ServerStream,
	clientStream grpc.ClientStream,
	state *bidiCaptureState,
	errCh chan<- error,
) {
	for {
		resp := dynamicpb.NewMessage(m.outputDesc)

		err := clientStream.RecvMsg(resp)
		if errors.Is(err, io.EOF) {
			errCh <- nil

			return
		}

		if err != nil {
			errCh <- err

			return
		}

		state.appendResponse(messageToMap(resp))

		if err = stream.SendMsg(resp); err != nil {
			errCh <- err

			return
		}
	}
}

func (m *grpcMocker) captureBidiResult(
	clientStream grpc.ClientStream,
	captureCtx captureRequestContext,
	requests []map[string]any,
	responses []map[string]any,
	firstErr error,
	secondErr error,
	recordDelay bool,
	elapsed time.Duration,
) {
	captureErr := selectCaptureError(firstErr, secondErr)
	captureErr = sanitizeCapturedStreamError(captureErr, len(responses) > 0)

	m.recordCapturedStub(
		func() *stuber.Stub {
			return proxycapture.BuildBidiStub(
				m.fullServiceName, m.methodName, captureCtx.sessionID,
				requests, captureCtx.headers, responses,
				responseHeadersFromClientStream(clientStream), captureErr,
			)
		},
		recordDelay, elapsed,
	)
}

func selectCaptureError(firstErr, secondErr error) error {
	if firstErr != nil {
		return firstErr
	}

	return secondErr
}

func sanitizeCapturedStreamError(err error, hasResponses bool) error {
	if err == nil {
		return nil
	}

	if !hasResponses {
		return err
	}

	if status.Code(err) == codes.Canceled {
		return nil
	}

	return err
}

type bidiCaptureState struct {
	mu        sync.Mutex
	requests  []map[string]any
	responses []map[string]any
}

func newBidiCaptureState() *bidiCaptureState {
	return &bidiCaptureState{
		requests:  make([]map[string]any, 0, proxyMessagesInitCap),
		responses: make([]map[string]any, 0, proxyMessagesInitCap),
	}
}

func (s *bidiCaptureState) appendRequest(req map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.requests = append(s.requests, req)
}

func (s *bidiCaptureState) appendResponse(resp map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.responses = append(s.responses, resp)
}

func (s *bidiCaptureState) snapshot() ([]map[string]any, []map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	requests := append([]map[string]any(nil), s.requests...)
	responses := append([]map[string]any(nil), s.responses...)

	return requests, responses
}
