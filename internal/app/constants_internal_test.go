package app

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// ConstantsTestSuite provides test suite for application constants.
type ConstantsTestSuite struct {
	suite.Suite
}

// TestConstants tests that all constants are properly defined.
func (s *ConstantsTestSuite) TestConstants() {
	s.Require().NotEmpty(excludedHeaders)
	s.Require().Contains(excludedHeaders, ":authority")
	s.Require().Contains(excludedHeaders, "content-type")
	s.Require().Contains(excludedHeaders, "grpc-accept-encoding")
	s.Require().Contains(excludedHeaders, "user-agent")
	s.Require().Contains(excludedHeaders, "accept-encoding")
}

// TestExcludedHeadersContent tests the content of excluded headers.
func (s *ConstantsTestSuite) TestExcludedHeadersContent() {
	expected := []string{
		":authority",
		"content-type",
		"grpc-accept-encoding",
		"user-agent",
		"accept-encoding",
	}

	s.Require().Len(excludedHeaders, len(expected))

	for _, h := range expected {
		s.Require().Contains(excludedHeaders, h)
	}
}

// TestLoggingFieldsFormat tests logging fields format constants.
func (s *ConstantsTestSuite) TestLoggingFieldsFormat() {
	// Test that logging fields are properly formatted strings
	s.Require().Equal("peer.address", LogFieldPeerAddress)
	s.Require().Equal("service", LogFieldService)
	s.Require().Equal("method", LogFieldMethod)
	s.Require().Equal("grpc.component", LogFieldComponent)
}

// TestConstantsTestSuite runs the constants test suite.
func TestConstantsTestSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(ConstantsTestSuite))
}
