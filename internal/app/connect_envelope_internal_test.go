package app

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadConnectFrame_EOF(t *testing.T) {
	t.Parallel()

	_, err := readConnectFrame(bytes.NewReader(nil))
	require.ErrorIs(t, err, io.EOF)
}

func TestReadConnectFrame_EmptyData(t *testing.T) {
	t.Parallel()

	// 5-byte header: flag=0, length=0
	header := []byte{0, 0, 0, 0, 0}

	frame, err := readConnectFrame(bytes.NewReader(header))
	require.NoError(t, err)
	require.Equal(t, byte(0), frame.flags)
	require.Nil(t, frame.data)
}

func TestReadConnectFrame_WithData(t *testing.T) {
	t.Parallel()

	payload := []byte("hello")
	header := []byte{0, 0, 0, 0, 5}

	frame, err := readConnectFrame(bytes.NewReader(append(header, payload...)))
	require.NoError(t, err)
	require.Equal(t, byte(0), frame.flags)
	require.Equal(t, payload, frame.data)
}

func TestReadConnectFrame_EndStreamFlag(t *testing.T) {
	t.Parallel()

	header := []byte{connectEnvelopeFlagEndStream, 0, 0, 0, 0}

	frame, err := readConnectFrame(bytes.NewReader(header))
	require.NoError(t, err)
	require.Equal(t, byte(connectEnvelopeFlagEndStream), frame.flags)
}

func TestReadConnectFrame_TruncatedHeader(t *testing.T) {
	t.Parallel()

	_, err := readConnectFrame(bytes.NewReader([]byte{0, 0, 0}))
	// io.ReadFull returns io.ErrUnexpectedEOF for a partial read,
	// which the caller surfaces directly as a protocol violation.
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

func TestReadConnectFrame_TruncatedBody(t *testing.T) {
	t.Parallel()

	// header says 10 bytes but only 3 are present
	header := []byte{0, 0, 0, 0, 10}
	_, err := readConnectFrame(bytes.NewReader(append(header, []byte("abc")...)))
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

// TestReadConnectFrame_OversizedBodyReturnsErrEnvelopeTooLarge guards
// against a malicious peer advertising an enormous length and forcing
// a huge allocation. The helper must reject such frames with
// ErrEnvelopeTooLarge before any memory is allocated. After this
// error the stream is poisoned and must not be reused.
func TestReadConnectFrame_OversizedBodyReturnsErrEnvelopeTooLarge(t *testing.T) {
	t.Parallel()

	// length = 1 GiB, body = 0 bytes (the reader closes immediately)
	header := []byte{0, 0x40, 0x00, 0x00, 0x00}

	_, err := readConnectFrame(bytes.NewReader(header))
	require.ErrorIs(t, err, ErrEnvelopeTooLarge)
}

// TestReadConnectFrame_ExactlyMaxSizeAccepted verifies that frames at
// exactly the maximum size are accepted.
func TestReadConnectFrame_ExactlyMaxSizeAccepted(t *testing.T) {
	t.Parallel()

	payload := make([]byte, connectEnvelopeMaxFrameSize)

	header := []byte{0, 0, 0, 0, 0}
	binary.BigEndian.PutUint32(header[1:5], uint32(connectEnvelopeMaxFrameSize))

	frame, err := readConnectFrame(bytes.NewReader(append(header, payload...)))
	require.NoError(t, err)
	require.Len(t, frame.data, connectEnvelopeMaxFrameSize)
}

// TestReadConnectFrame_OneByteOverMaxSizeRejected verifies the boundary.
func TestReadConnectFrame_OneByteOverMaxSizeRejected(t *testing.T) {
	t.Parallel()

	header := []byte{0, 0, 0, 0, 0}
	binary.BigEndian.PutUint32(header[1:5], uint32(connectEnvelopeMaxFrameSize+1))

	_, err := readConnectFrame(bytes.NewReader(header))
	require.ErrorIs(t, err, ErrEnvelopeTooLarge)
}

func TestWriteConnectFrame_EmptyData(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, writeConnectFrame(&buf, nil, false))
	require.Equal(t, []byte{0, 0, 0, 0, 0}, buf.Bytes())
}

func TestWriteConnectFrame_WithData(t *testing.T) {
	t.Parallel()

	payload := []byte("hello")

	var buf bytes.Buffer
	require.NoError(t, writeConnectFrame(&buf, payload, false))

	expected := make([]byte, 0, connectEnvelopeHeaderSize+len(payload))
	expected = append(expected, 0)
	lengthBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBuf, uint32(len(payload))) //nolint:gosec
	expected = append(expected, lengthBuf...)
	expected = append(expected, payload...)

	require.Equal(t, expected, buf.Bytes())
}

func TestWriteConnectFrame_EndStream(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, writeConnectFrame(&buf, nil, true))
	require.Equal(t, byte(connectEnvelopeFlagEndStream), buf.Bytes()[0])
}

func TestReadWriteConnectFrame_Roundtrip(t *testing.T) {
	t.Parallel()

	payloads := [][]byte{
		[]byte("first"),
		[]byte("second message"),
		nil, // empty frame
	}

	var buf bytes.Buffer

	for i, p := range payloads {
		endStream := i == len(payloads)-1
		require.NoError(t, writeConnectFrame(&buf, p, endStream))
	}

	for i, p := range payloads {
		frame, err := readConnectFrame(&buf)
		require.NoError(t, err)
		require.Equal(t, p, frame.data)

		if i == len(payloads)-1 {
			require.Equal(t, byte(connectEnvelopeFlagEndStream), frame.flags)
		} else {
			require.Equal(t, byte(0), frame.flags)
		}
	}

	_, err := readConnectFrame(&buf)
	require.ErrorIs(t, err, io.EOF)
}
