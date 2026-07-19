package app

import (
	"encoding/binary"
	"errors"
	"io"
)

// Connect RPC envelope framing constants.
// Reference: https://connectrpc.com/docs/protocol/
const (
	connectEnvelopeFlagEndStream = 0b00000010
	ConnectEnvelopeHeaderSize    = 5

	// connectEnvelopeMaxFrameSize caps the per-message payload a peer may
	// advertise. The Connect spec does not define an upper bound; we
	// default to 4 MiB which matches httputil.MaxBodyBytes and is
	// generous for typical RPC payloads. Larger values trigger an
	// explicit error instead of silently allocating gigabytes.
	connectEnvelopeMaxFrameSize = 4 << 20
)

// ErrEnvelopeTooLarge is returned by readConnectFrame when the peer
// advertises a payload larger than connectEnvelopeMaxFrameSize.
// After this error the reader stream is poisoned (the header was
// consumed but the body was not) and must not be used for further
// reads.
var ErrEnvelopeTooLarge = errors.New("connect envelope payload exceeds maximum size")

// connectError is the JSON body of a Connect RPC error response.
// See https://connectrpc.com/docs/protocol/#error-end-stream
type connectError struct {
	Code    string           `json:"code"`
	Message string           `json:"message"`
	Details []map[string]any `json:"details"`
}

type connectFrame struct {
	flags byte
	data  []byte
}

// readConnectFrame reads a single Connect RPC envelope from r.
// Returns io.EOF if the stream ended cleanly (no partial data) and
// io.ErrUnexpectedEOF if the peer closed mid-frame (protocol violation).
func readConnectFrame(r io.Reader) (connectFrame, error) {
	var header [ConnectEnvelopeHeaderSize]byte

	_, err := io.ReadFull(r, header[:])
	if errors.Is(err, io.EOF) {
		return connectFrame{}, io.EOF
	}

	if err != nil {
		return connectFrame{}, err
	}

	flags := header[0]
	length := binary.BigEndian.Uint32(header[1:5])

	if length == 0 {
		return connectFrame{flags: flags, data: nil}, nil
	}

	if length > connectEnvelopeMaxFrameSize {
		return connectFrame{}, ErrEnvelopeTooLarge
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return connectFrame{}, err
	}

	return connectFrame{flags: flags, data: data}, nil
}

// writeConnectFrame writes a single Connect RPC envelope to w. The endStream
// flag (0x02) is set when endStream is true (used to signal the end of
// server streaming).
func writeConnectFrame(w io.Writer, data []byte, endStream bool) error {
	var header [ConnectEnvelopeHeaderSize]byte

	flags := byte(0)
	if endStream {
		flags |= connectEnvelopeFlagEndStream
	}

	header[0] = flags
	binary.BigEndian.PutUint32(header[1:5], uint32(len(data))) //nolint:gosec

	if _, err := w.Write(header[:]); err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}

	if _, err := w.Write(data); err != nil {
		return err
	}

	return nil
}
