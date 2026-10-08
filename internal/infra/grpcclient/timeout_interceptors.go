package grpcclient

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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

		longLived := desc != nil && desc.ServerStreams

		streamCtx, cancel := newStreamContext(ctx, timeout, longLived)

		var watchdog *time.Timer
		if longLived {
			watchdog = time.AfterFunc(timeout, cancel)
		}

		clientStream, err := streamer(streamCtx, desc, cc, method, opts...)
		if err != nil {
			if watchdog != nil {
				watchdog.Stop()
			}

			cancel()

			return nil, err
		}

		return &wrappedClientStream{
			ClientStream:        clientStream,
			cancel:              cancel,
			watchdog:            watchdog,
			cancelOnRecvSuccess: !longLived,
		}, nil
	}
}

func newStreamContext(
	ctx context.Context,
	timeout time.Duration,
	longLived bool,
) (context.Context, context.CancelFunc) {
	if longLived {
		return context.WithCancel(ctx)
	}

	return context.WithTimeout(ctx, timeout)
}

type wrappedClientStream struct {
	grpc.ClientStream

	cancel     context.CancelFunc
	cancelOnce sync.Once

	watchdog     *time.Timer
	watchdogOnce sync.Once

	cancelOnRecvSuccess bool
}

func (w *wrappedClientStream) RecvMsg(m any) error {
	err := w.ClientStream.RecvMsg(m)
	if err != nil {
		w.stopWatchdog()
		w.cancelContext()

		return err
	}

	w.stopWatchdog()

	if w.cancelOnRecvSuccess {
		w.cancelContext()
	}

	return err
}

func (w *wrappedClientStream) Header() (metadata.MD, error) {
	md, err := w.ClientStream.Header()
	if err == nil {
		w.stopWatchdog()
	}

	return md, err //nolint:wrapcheck // transparent passthrough.
}

func (w *wrappedClientStream) CloseSend() error {
	return w.ClientStream.CloseSend()
}

func (w *wrappedClientStream) stopWatchdog() {
	if w.watchdog == nil {
		return
	}

	w.watchdogOnce.Do(func() {
		w.watchdog.Stop()
	})
}

func (w *wrappedClientStream) cancelContext() {
	w.cancelOnce.Do(w.cancel)
}
