package grpcclient

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc"
)

// UnaryTimeoutInterceptor sets timeout for unary gRPC client calls when timeout > 0.
// Existing deadlines are preserved.
func UnaryTimeoutInterceptor(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req,
		reply any,
		conn *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if timeout > 0 {
			if _, ok := ctx.Deadline(); !ok {
				var cancel context.CancelFunc

				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}
		}

		return invoker(ctx, method, req, reply, conn, opts...)
	}
}

// StreamTimeoutInterceptor sets timeout for stream initialization when timeout > 0.
// Existing deadlines are preserved.
func StreamTimeoutInterceptor(timeout time.Duration) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		if timeout <= 0 {
			return streamer(ctx, desc, cc, method, opts...)
		}

		if _, ok := ctx.Deadline(); ok {
			return streamer(ctx, desc, cc, method, opts...)
		}

		streamCtx, cancel := context.WithTimeout(ctx, timeout)

		clientStream, err := streamer(streamCtx, desc, cc, method, opts...)
		if err != nil {
			cancel()

			return nil, err
		}

		return &wrappedClientStream{
			ClientStream:        clientStream,
			cancel:              cancel,
			cancelOnRecvSuccess: desc == nil || !desc.ServerStreams,
		}, nil
	}
}

type wrappedClientStream struct {
	grpc.ClientStream

	cancel     context.CancelFunc
	cancelOnce sync.Once

	cancelOnRecvSuccess bool
}

func (w *wrappedClientStream) RecvMsg(m any) error {
	err := w.ClientStream.RecvMsg(m)
	if err != nil {
		w.cancelContext()

		return err
	}

	if w.cancelOnRecvSuccess {
		w.cancelContext()
	}

	return err
}

func (w *wrappedClientStream) CloseSend() error {
	return w.ClientStream.CloseSend()
}

func (w *wrappedClientStream) cancelContext() {
	w.cancelOnce.Do(w.cancel)
}
