//go:build windows && cgo

// Package export carries the C entry points a plugin DLL must expose.
package export

import "C"

import (
	"context"
	"unsafe"

	"github.com/bavix/gripmock/v3/pkg/plugins"
	"github.com/bavix/gripmock/v3/pkg/plugins/cshared"
)

// Use registers the plugin the exported entry points answer for.
func Use(register func(plugins.Registry)) {
	cshared.Use(register)
}

//export GripMockPluginDescribe
func GripMockPluginDescribe(buf *C.char, capacity C.int) C.int {
	return write(cshared.Describe(), buf, capacity)
}

//export GripMockPluginCall
func GripMockPluginCall(name *C.char, args *C.char, buf *C.char, capacity C.int) C.int {
	return write(cshared.Call(context.Background(), C.GoString(name), C.GoString(args)), buf, capacity)
}

// write fills the buffer the server owns and reports the length the answer
// needs, so a short buffer is reported instead of overflowing.
func write(out string, buf *C.char, capacity C.int) C.int {
	if buf != nil && int(capacity) >= len(out) {
		//nolint:gosec
		copy(unsafe.Slice((*byte)(unsafe.Pointer(buf)), int(capacity)), out)
	}

	return C.int(len(out))
}
