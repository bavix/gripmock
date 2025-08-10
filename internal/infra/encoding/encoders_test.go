package encoding_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/encoding"
)

func TestNewBase64Utils(t *testing.T) {
	t.Parallel()

	utils := encoding.NewBase64Utils()
	require.NotNil(t, utils)
}

func TestBase64Utils_StringToBase64(t *testing.T) {
	t.Parallel()

	utils := encoding.NewBase64Utils()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    "hello world",
			expected: "aGVsbG8gd29ybGQ=",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "special characters",
			input:    "Hello, World! 🌍",
			expected: "SGVsbG8sIFdvcmxkISDwn4yN",
		},
		{
			name:     "numbers and symbols",
			input:    "12345!@#$%",
			expected: "MTIzNDUhQCMkJQ==",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := utils.StringToBase64(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestBase64Utils_BytesToBase64(t *testing.T) {
	t.Parallel()

	utils := encoding.NewBase64Utils()

	testCases := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "simple bytes",
			input:    []byte("hello world"),
			expected: "aGVsbG8gd29ybGQ=",
		},
		{
			name:     "empty bytes",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "nil bytes",
			input:    nil,
			expected: "",
		},
		{
			name:     "binary data",
			input:    []byte{0x01, 0x02, 0x03, 0x04, 0xFF},
			expected: "AQIDBP8=",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := utils.BytesToBase64(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestNewUUIDUtils(t *testing.T) {
	t.Parallel()

	utils := encoding.NewUUIDUtils()
	require.NotNil(t, utils)
}

func TestUUIDUtils_UUIDToBase64(t *testing.T) {
	t.Parallel()

	utils := encoding.NewUUIDUtils()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid UUID v4",
			input:    "77465064-a0ce-48a3-b7e4-d50f88e55093",
			expected: "d0ZQZKDOSKO35NUPiOVQkw==",
		},
		{
			name:     "nil UUID",
			input:    "00000000-0000-0000-0000-000000000000",
			expected: "AAAAAAAAAAAAAAAAAAAAAA==",
		},
		{
			name:     "max UUID",
			input:    "ffffffff-ffff-ffff-ffff-ffffffffffff",
			expected: "/////////////////////w==",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := utils.UUIDToBase64(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}

	t.Run("invalid UUID", func(t *testing.T) {
		t.Parallel()

		require.Panics(t, func() {
			utils.UUIDToBase64("invalid-uuid")
		})
	})
}

func TestUUIDUtils_UUIDToBytes(t *testing.T) {
	t.Parallel()

	utils := encoding.NewUUIDUtils()

	testCases := []struct {
		name     string
		input    string
		expected []byte
	}{
		{
			name:  "valid UUID v4",
			input: "77465064-a0ce-48a3-b7e4-d50f88e55093",
			expected: []byte{
				0x77, 0x46, 0x50, 0x64, 0xa0, 0xce, 0x48, 0xa3,
				0xb7, 0xe4, 0xd5, 0x0f, 0x88, 0xe5, 0x50, 0x93,
			},
		},
		{
			name:     "nil UUID",
			input:    "00000000-0000-0000-0000-000000000000",
			expected: make([]byte, 16), // all zeros
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := utils.UUIDToBytes(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}

	t.Run("invalid UUID", func(t *testing.T) {
		t.Parallel()

		require.Panics(t, func() {
			utils.UUIDToBytes("invalid-uuid")
		})
	})
}

func TestUUIDUtils_UUIDToInt64(t *testing.T) {
	t.Parallel()

	utils := encoding.NewUUIDUtils()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid UUID v4",
			input:    "e351220b-4847-42f5-8abb-c052b87ff2d4",
			expected: `{"high":-773977811204288029,"low":-3102276763665777782}`,
		},
		{
			name:     "nil UUID",
			input:    "00000000-0000-0000-0000-000000000000",
			expected: `{"high":0,"low":0}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := utils.UUIDToInt64(tc.input)
			require.JSONEq(t, tc.expected, result)
		})
	}

	t.Run("invalid UUID", func(t *testing.T) {
		t.Parallel()

		require.Panics(t, func() {
			utils.UUIDToInt64("invalid-uuid")
		})
	})
}

func TestNewConversionUtils(t *testing.T) {
	t.Parallel()

	utils := encoding.NewConversionUtils()
	require.NotNil(t, utils)
}

func TestConversionUtils_StringToBytes(t *testing.T) {
	t.Parallel()

	utils := encoding.NewConversionUtils()

	testCases := []struct {
		name     string
		input    string
		expected []byte
	}{
		{
			name:     "simple string",
			input:    "hello",
			expected: []byte("hello"),
		},
		{
			name:     "empty string",
			input:    "",
			expected: []byte{},
		},
		{
			name:     "unicode string",
			input:    "Hello, World!",
			expected: []byte("Hello, World!"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := utils.StringToBytes(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestNewTemplateUtils(t *testing.T) {
	t.Parallel()

	utils := encoding.NewTemplateUtils()
	require.NotNil(t, utils)
	require.NotNil(t, utils.Base64)
	require.NotNil(t, utils.UUID)
	require.NotNil(t, utils.Conversion)
}

func TestTemplateUtils_Integration(t *testing.T) {
	t.Parallel()

	utils := encoding.NewTemplateUtils()

	t.Run("all utils work together", func(t *testing.T) {
		t.Parallel()

		// Test Base64 utils
		base64Result := utils.Base64.StringToBase64("hello")
		require.Equal(t, "aGVsbG8=", base64Result)

		// Test UUID utils
		uuidResult := utils.UUID.UUIDToBase64("77465064-a0ce-48a3-b7e4-d50f88e55093")
		require.Equal(t, "d0ZQZKDOSKO35NUPiOVQkw==", uuidResult)

		// Test Conversion utils
		bytesResult := utils.Conversion.StringToBytes("test")
		require.Equal(t, []byte("test"), bytesResult)

		// Test chaining operations
		chainedResult := utils.Base64.BytesToBase64(utils.Conversion.StringToBytes("chain"))
		require.Equal(t, "Y2hhaW4=", chainedResult)
	})
}
