package app

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/peer"
)

// GRPCServerTestSuite provides test suite for gRPC server internal functionality.
type GRPCServerTestSuite struct {
	suite.Suite
}

// TestSplitMethodName tests method name splitting functionality.
func (s *GRPCServerTestSuite) TestSplitMethodName() {
	tests := []struct {
		name           string
		fullMethod     string
		expectedSvc    string
		expectedMethod string
	}{
		{
			name:           "simple service method",
			fullMethod:     "/TestService/TestMethod",
			expectedSvc:    "TestService",
			expectedMethod: "TestMethod",
		},
		{
			name:           "packaged service method",
			fullMethod:     "/test.v1.TestService/TestMethod",
			expectedSvc:    "test.v1.TestService",
			expectedMethod: "TestMethod",
		},
		{
			name:           "deeply nested package",
			fullMethod:     "/com.example.api.v1.TestService/TestMethod",
			expectedSvc:    "com.example.api.v1.TestService",
			expectedMethod: "TestMethod",
		},
		{
			name:           "no package",
			fullMethod:     "/Service/Method",
			expectedSvc:    "Service",
			expectedMethod: "Method",
		},
		{
			name:           "invalid format",
			fullMethod:     "invalid",
			expectedSvc:    "unknown",
			expectedMethod: "unknown",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			svc, method := splitMethodName(tt.fullMethod)
			s.Equal(tt.expectedSvc, svc)
			s.Equal(tt.expectedMethod, method)
		})
	}
}

// TestGetPeerAddress tests peer address extraction functionality.
func (s *GRPCServerTestSuite) TestGetPeerAddress() {
	tests := []struct {
		name     string
		peer     *peer.Peer
		expected string
	}{
		{
			name:     "nil peer",
			peer:     nil,
			expected: "unknown",
		},
		{
			name: "peer with address",
			peer: &peer.Peer{
				Addr: &mockAddr{addr: "127.0.0.1:12345"},
			},
			expected: "127.0.0.1:12345",
		},
		{
			name: "peer with nil address",
			peer: &peer.Peer{
				Addr: nil,
			},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := getPeerAddress(tt.peer)
			s.Equal(tt.expected, result)
		})
	}
}

// mockAddr is a mock implementation of net.Addr for testing.
type mockAddr struct {
	addr string
}

func (m *mockAddr) Network() string {
	return "tcp"
}

func (m *mockAddr) String() string {
	return m.addr
}

// TestGRPCServerTestSuite runs the gRPC server test suite.
func TestGRPCServerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(GRPCServerTestSuite))
}
