package app

import (
	"io"
	"sync"

	"github.com/cockroachdb/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
)

//nolint:cyclop,funlen,wsl_v5
func (m *grpcMocker) proxyServerStream(stream grpc.ServerStream, route *proxyroutes.Route, capture bool) error {
	proxyCtx, cancel := route.WithTimeout(proxyroutes.ForwardIncomingMetadata(stream.Context()))
	defer cancel()

	desc := &grpc.StreamDesc{ServerStreams: true, ClientStreams: false}
	clientStream, err := route.Conn.NewStream(proxyCtx, desc, m.fullMethod)
	if err != nil {
		return err
	}

	req := dynamicpb.NewMessage(m.inputDesc)
	if err = stream.RecvMsg(req); err != nil {
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

	for {
		resp := dynamicpb.NewMessage(m.outputDesc)
		err = clientStream.RecvMsg(resp)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			if capture {
				m.recordCapturedServerStreamStub(convertToMap(req), responses, m.sessionFromContext(stream.Context()))
			}

			return err
		}

		responses = append(responses, convertToMap(resp))

		if err = stream.SendMsg(resp); err != nil {
			return err
		}
	}

	if trailer := clientStream.Trailer(); len(trailer) > 0 {
		stream.SetTrailer(trailer)
	}

	if capture {
		m.recordCapturedServerStreamStub(convertToMap(req), responses, m.sessionFromContext(stream.Context()))
	}

	return nil
}

//nolint:cyclop,funlen,wsl_v5
func (m *grpcMocker) proxyClientStream(stream grpc.ServerStream, route *proxyroutes.Route, capture bool) error {
	proxyCtx, cancel := route.WithTimeout(proxyroutes.ForwardIncomingMetadata(stream.Context()))
	defer cancel()

	desc := &grpc.StreamDesc{ServerStreams: false, ClientStreams: true}
	clientStream, err := route.Conn.NewStream(proxyCtx, desc, m.fullMethod)
	if err != nil {
		return err
	}

	requests := make([]map[string]any, 0, proxyMessagesInitCap)

	for {
		req := dynamicpb.NewMessage(m.inputDesc)
		err = stream.RecvMsg(req)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return err
		}

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
			m.recordCapturedClientStreamStub(requests, nil, err, m.sessionFromContext(stream.Context()))
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
		m.recordCapturedClientStreamStub(requests, convertToMap(resp), nil, m.sessionFromContext(stream.Context()))
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
			responses = append(responses, convertToMap(resp))
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
		m.recordCapturedBidiStub(requests, responses, m.sessionFromContext(stream.Context()))
	}

	if firstErr != nil {
		return firstErr
	}

	if secondErr != nil {
		return secondErr
	}

	return nil
}
