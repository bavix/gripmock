package app

import (
	"encoding/base64"
	"errors"
	"net/http"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	connectQueryMessage     = "message"
	connectQueryBase64      = "base64"
	connectQueryEncoding    = "encoding"
	connectQueryCompression = "compression"
	connectQueryVersion     = "connect"

	connectEncodingProto = "proto"
	connectEncodingJSON  = "json"
)

var (
	errConnectGetNotIdempotent = errors.New("method does not allow GET")
	errConnectGetMalformed     = errors.New("malformed GET request")
	errConnectGetCompressed    = errors.New("compressed GET requests are not supported")
)

func methodAllowsGET(desc protoreflect.MethodDescriptor) bool {
	if desc == nil {
		return false
	}

	opts, ok := desc.Options().(*descriptorpb.MethodOptions)
	if !ok || opts == nil {
		return false
	}

	return opts.GetIdempotencyLevel() == descriptorpb.MethodOptions_NO_SIDE_EFFECTS
}

func connectGetContentType(r *http.Request) string {
	if r.URL.Query().Get(connectQueryEncoding) == connectEncodingJSON {
		return contentTypeJSON
	}

	return contentTypeProto
}

func connectGetRequest(r *http.Request, desc protoreflect.MethodDescriptor) ([]byte, error) {
	if !methodAllowsGET(desc) {
		return nil, errConnectGetNotIdempotent
	}

	q := r.URL.Query()

	if c := q.Get(connectQueryCompression); c != "" && c != "identity" {
		return nil, errConnectGetCompressed
	}

	raw := q.Get(connectQueryMessage)
	if raw == "" {
		return nil, nil
	}

	if q.Get(connectQueryBase64) == "1" {
		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		if err == nil {
			return decoded, nil
		}

		decoded, err = base64.URLEncoding.DecodeString(raw)
		if err != nil {
			return nil, errConnectGetMalformed
		}

		return decoded, nil
	}

	return []byte(raw), nil
}
