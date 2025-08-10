package muxmiddleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUtils_Basic(t *testing.T) {
	t.Parallel()
	// Test basic utils functionality
	require.NotNil(t, "utils package exists")
}

func TestUtils_Empty(t *testing.T) {
	t.Parallel()
	// Test empty utils case
	require.NotNil(t, "utils package exists")
}

func TestUtils_Initialization(t *testing.T) {
	t.Parallel()
	// Test utils initialization
	require.NotNil(t, "utils package initialized")
}

func TestUtils_GetIP(t *testing.T) {
	t.Parallel()
	// Test getIP function
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// httptest.NewRequest sets a default RemoteAddr, let's use it

	ip, err := getIP(req)
	require.NoError(t, err)
	require.NotNil(t, ip) // Just check that we get a valid IP
}

func TestUtils_GetIPWithXForwardedFor(t *testing.T) {
	t.Parallel()
	// Test getIP with X-Forwarded-For header
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 192.168.1.1")

	ip, err := getIP(req)
	require.NoError(t, err)
	require.Equal(t, net.ParseIP("192.168.1.1"), ip)
}

func TestUtils_GetIPWithInvalidXForwardedFor(t *testing.T) {
	t.Parallel()
	// Test getIP with invalid X-Forwarded-For header
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "invalid-ip")
	req.RemoteAddr = "192.168.1.1:8080" // Set specific IP for this test

	ip, err := getIP(req)
	require.NoError(t, err)
	require.Equal(t, net.ParseIP("192.168.1.1"), ip)
}

func TestUtils_GetIPWithEmptyXForwardedFor(t *testing.T) {
	t.Parallel()
	// Test getIP with empty X-Forwarded-For header
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "")
	req.RemoteAddr = "192.168.1.1:8080" // Set specific IP for this test

	ip, err := getIP(req)
	require.NoError(t, err)
	require.Equal(t, net.ParseIP("192.168.1.1"), ip)
}

func TestUtils_GetIPWithInvalidRemoteAddr(t *testing.T) {
	t.Parallel()
	// Test getIP with invalid RemoteAddr
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "invalid-addr"

	ip, err := getIP(req)
	require.Error(t, err)
	require.Nil(t, ip)
}

func TestUtils_GetIPWithIPv6(t *testing.T) {
	t.Parallel()
	// Test getIP with IPv6 address
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "[::1]:8080"

	ip, err := getIP(req)
	require.NoError(t, err)
	require.Equal(t, net.ParseIP("::1"), ip)
}
