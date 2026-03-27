package app

import (
	"io"
	"sync"

	"github.com/cockroachdb/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
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

	for {
		resp := dynamicpb.NewMessage(m.outputDesc)
		err = clientStream.RecvMsg(resp)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			if capture {
				m.recordCapturedServerStreamStub(
					requestData,
					captureCtx.headers,
					responses,
					responseHeadersFromClientStream(clientStream),
					err,
					captureCtx.sessionID,
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
		m.recordCapturedServerStreamStub(
			requestData,
			captureCtx.headers,
			responses,
			responseHeadersFromClientStream(clientStream),
			nil,
			captureCtx.sessionID,
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
	proxyCtx, cancel := route.WithTimeout(proxyroutes.ForwardIncomingMetadata(stream.Context()))
	defer cancel()

	desc := &grpc.StreamDesc{ServerStreams: false, ClientStreams: true}

	clientStream, err := route.Conn.NewStream(proxyCtx, desc, m.fullMethod)
	if err != nil {
		return err
	}

	requests := make([]map[string]any, 0, proxyMessagesInitCap)
	captureCtx := m.newCaptureRequestContext(stream.Context())

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
			m.recordCapturedClientStreamStub(
				requests,
				captureCtx.headers,
				nil,
				responseHeadersFromClientStream(clientStream),
				err,
				captureCtx.sessionID,
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
		m.recordCapturedClientStreamStub(
			requests,
			captureCtx.headers,
			messageToMap(resp),
			responseHeadersFromClientStream(clientStream),
			nil,
			captureCtx.sessionID,
		)
	}

	return nil
}

//nolint:cyclop,funlen,wsl_v5
func (m *grpcMocker) proxyBidiStream(stream grpc.ServerStream, route *proxyroutes.Route, capture bool) error {
	proxyCtx, cancel := route.WithTimeout(proxyroutes.ForwardIncomingMetadata(stream.Context()))
	defer cancel()

	desc := &grpc.StreamDesc{ServerStreams: true, ClientStreams: true}
	clientStream, err := route.Conn.NewStream(proxyCtx, desc, m.fullMethod)
	if err != nil {
		return err
	}

	var (
		mu        sync.Mutex
		requests  = make([]map[string]any, 0, proxyMessagesInitCap)
		responses = make([]map[string]any, 0, proxyMessagesInitCap)
	)

	captureCtx := m.newCaptureRequestContext(stream.Context())

	errCh := make(chan error, proxyErrChanCap)

	go func() {
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

			mu.Lock()
			requests = append(requests, convertToMap(req))
			mu.Unlock()

			if err = clientStream.SendMsg(req); err != nil {
				errCh <- err

				return
			}
		}
	}()

	go func() {
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

			mu.Lock()
			responses = append(responses, messageToMap(resp))
			mu.Unlock()

			if err = stream.SendMsg(resp); err != nil {
				errCh <- err

				return
			}
		}
	}()

	firstErr := <-errCh
	secondErr := <-errCh

	if trailer := clientStream.Trailer(); len(trailer) > 0 {
		stream.SetTrailer(trailer)
	}

	if capture {
		captureErr := selectCaptureError(firstErr, secondErr)
		captureErr = sanitizeCapturedStreamError(captureErr, len(responses) > 0)

		m.recordCapturedBidiStub(
			requests,
			captureCtx.headers,
			responses,
			responseHeadersFromClientStream(clientStream),
			captureErr,
			captureCtx.sessionID,
		)
	}

	if firstErr != nil {
		return firstErr
	}

	if secondErr != nil {
		return secondErr
	}

	return nil
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
