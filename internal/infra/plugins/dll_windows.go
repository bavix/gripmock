//go:build windows

package plugins

import (
	"context"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
)

const (
	describeSymbol = "GripMockPluginDescribe"
	callSymbol     = "GripMockPluginCall"

	initialBufferSize = 16 << 10
	maxBufferSize     = 16 << 20
)

func loadDLL(ctx context.Context, reg pkgplugins.Registry, path string) error {
	dll := syscall.NewLazyDLL(path)

	err := dll.Load()
	if err != nil {
		return fmt.Errorf("load dll: %w", err)
	}

	describe, call := dll.NewProc(describeSymbol), dll.NewProc(callSymbol)

	for _, proc := range []*syscall.LazyProc{describe, call} {
		err = proc.Find()
		if err != nil {
			return fmt.Errorf("missing entry point: %w", err)
		}
	}

	manifest, err := read(func(buf []byte) uintptr {
		//nolint:gosec
		size, _, _ := describe.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))

		runtime.KeepAlive(buf)

		return size
	})
	if err != nil {
		return err
	}

	plug := &dllPlugin{path: path, invoke: invoker(call)}

	return plug.register(ctx, reg, manifest)
}

func invoker(call *syscall.LazyProc) func(string, string) (string, error) {
	return func(name string, argsJSON string) (string, error) {
		namePtr, err := syscall.BytePtrFromString(name)
		if err != nil {
			return "", fmt.Errorf("encode name: %w", err)
		}

		argsPtr, err := syscall.BytePtrFromString(argsJSON)
		if err != nil {
			return "", fmt.Errorf("encode args: %w", err)
		}

		return read(func(buf []byte) uintptr {
			//nolint:gosec
			size, _, _ := call.Call(
				uintptr(unsafe.Pointer(namePtr)),
				uintptr(unsafe.Pointer(argsPtr)),
				uintptr(unsafe.Pointer(&buf[0])),
				uintptr(len(buf)),
			)

			runtime.KeepAlive(namePtr)
			runtime.KeepAlive(argsPtr)
			runtime.KeepAlive(buf)

			return size
		})
	}
}

// read calls fn with a buffer the plugin fills. The plugin answers with the
// length it needs, so a buffer that was too small is retried once, grown.
func read(fn func([]byte) uintptr) (string, error) {
	buf := make([]byte, initialBufferSize)

	for range 2 {
		//nolint:gosec
		size := int32(uint32(fn(buf)))

		switch {
		case size < 0 || int(size) > maxBufferSize:
			return "", errPlugin
		case int(size) <= len(buf):
			return string(buf[:size]), nil
		}

		buf = make([]byte, size)
	}

	return "", errPlugin
}
