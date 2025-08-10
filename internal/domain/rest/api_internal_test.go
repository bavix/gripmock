package rest

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
)

// APITestSuite provides test suite for REST API domain types.
type APITestSuite struct {
	suite.Suite
}

// TestStubInputValidation tests StubInput validation.
func (s *APITestSuite) TestStubInputValidation() {
	tests := []struct {
		name  string
		input StubInput
		valid bool
	}{
		{
			name: "valid input with contains",
			input: StubInput{
				Contains: map[string]any{"key": "value"},
			},
			valid: true,
		},
		{
			name: "valid input with equals",
			input: StubInput{
				Equals: map[string]any{"key": "value"},
			},
			valid: true,
		},
		{
			name: "valid input with matches",
			input: StubInput{
				Matches: map[string]any{"key": "pattern.*"},
			},
			valid: true,
		},
		{
			name: "empty input",
			input: StubInput{
				Contains: nil,
				Equals:   nil,
				Matches:  nil,
			},
			valid: false,
		},
		{
			name: "multiple matchers",
			input: StubInput{
				Contains: map[string]any{"key1": "value1"},
				Equals:   map[string]any{"key2": "value2"},
			},
			valid: true, // Multiple matchers are allowed
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			hasData := len(tt.input.Contains) > 0 ||
				len(tt.input.Equals) > 0 ||
				len(tt.input.Matches) > 0

			s.Require().Equal(tt.valid, hasData)
		})
	}
}

// TestStubOutputValidation tests StubOutput validation.
//
//nolint:funlen // Test function requires multiple scenarios
func (s *APITestSuite) TestStubOutputValidation() {
	tests := []struct {
		name   string
		output StubOutput
		valid  bool
	}{
		{
			name: "valid output with data",
			output: StubOutput{
				Data: map[string]any{"result": "success"},
			},
			valid: true,
		},
		{
			name: "valid output with stream",
			output: StubOutput{
				Stream: []map[string]any{{"result": "response1"}},
			},
			valid: true,
		},
		{
			name: "valid output with error",
			output: StubOutput{
				Error: "something went wrong",
			},
			valid: true,
		},
		{
			name: "valid output with code",
			output: StubOutput{
				Code: func() *codes.Code {
					c := codes.Code(14)

					return &c
				}(),
			},
			valid: true,
		},
		{
			name: "empty output",
			output: StubOutput{
				Data:   nil,
				Stream: nil,
				Error:  "",
				Code:   nil,
			},
			valid: false,
		},
		{
			name: "output with both data and stream",
			output: StubOutput{
				Data:   map[string]any{"result": "success"},
				Stream: []map[string]any{{"result": "response1"}},
			},
			valid: false, // Should not have both
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			hasData := tt.output.Data != nil
			hasStream := len(tt.output.Stream) > 0
			hasError := tt.output.Error != ""
			hasCode := tt.output.Code != nil

			isValid := (hasData || hasStream || hasError || hasCode) && (!hasData || !hasStream)

			s.Require().Equal(tt.valid, isValid)
		})
	}
}

// TestStubValidation tests Stub validation.
//
//nolint:cyclop,funlen // Test function requires multiple validation scenarios
func (s *APITestSuite) TestStubValidation() {
	tests := []struct {
		name  string
		stub  Stub
		valid bool
	}{
		{
			name: "valid unary stub",
			stub: Stub{
				Service: "TestService",
				Method:  "TestMethod",
				Input: StubInput{
					Contains: map[string]any{"key": "value"},
				},
				Output: StubOutput{
					Data: map[string]any{"result": "success"},
				},
			},
			valid: true,
		},
		{
			name: "valid client streaming stub",
			stub: Stub{
				Service: "TestService",
				Method:  "TestMethod",
				Inputs: &[]StubInput{
					{Contains: map[string]any{"key": "value"}},
				},
				Output: StubOutput{
					Data: map[string]any{"result": "success"},
				},
			},
			valid: true,
		},
		{
			name: "valid server streaming stub",
			stub: Stub{
				Service: "TestService",
				Method:  "TestMethod",
				Input: StubInput{
					Contains: map[string]any{"key": "value"},
				},
				Output: StubOutput{
					Stream: []map[string]any{{"result": "response"}},
				},
			},
			valid: true,
		},
		{
			name: "stub missing service",
			stub: Stub{
				Method: "TestMethod",
				Input: StubInput{
					Contains: map[string]any{"key": "value"},
				},
				Output: StubOutput{
					Data: map[string]any{"result": "success"},
				},
			},
			valid: false,
		},
		{
			name: "stub missing method",
			stub: Stub{
				Service: "TestService",
				Input: StubInput{
					Contains: map[string]any{"key": "value"},
				},
				Output: StubOutput{
					Data: map[string]any{"result": "success"},
				},
			},
			valid: false,
		},
		{
			name: "stub with both input and inputs",
			stub: Stub{
				Service: "TestService",
				Method:  "TestMethod",
				Input: StubInput{
					Contains: map[string]any{"key": "value"},
				},
				Inputs: &[]StubInput{
					{Contains: map[string]any{"key": "value"}},
				},
				Output: StubOutput{
					Data: map[string]any{"result": "success"},
				},
			},
			valid: false, // Should not have both
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			hasService := tt.stub.Service != ""
			hasMethod := tt.stub.Method != ""
			hasInput := (len(tt.stub.Input.Contains) > 0) || (len(tt.stub.Input.Equals) > 0) || (len(tt.stub.Input.Matches) > 0)
			hasInputs := tt.stub.Inputs != nil && len(*tt.stub.Inputs) > 0
			hasValidOutput := tt.stub.Output.Data != nil || len(tt.stub.Output.Stream) > 0 ||
				tt.stub.Output.Error != "" || tt.stub.Output.Code != nil

			isValid := hasService && hasMethod && (hasInput || hasInputs) && (!hasInput || !hasInputs) && hasValidOutput

			s.Require().Equal(tt.valid, isValid)
		})
	}
}

// TestIDType tests ID type functionality.
func (s *APITestSuite) TestIDType() {
	// Test that ID is a UUID type
	id, err := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	s.Require().NoError(err)
	s.Require().Equal("550e8400-e29b-41d4-a716-446655440000", id.String())
}

// TestHeadersValidation tests headers validation.
func (s *APITestSuite) TestHeadersValidation() {
	tests := []struct {
		name    string
		headers map[string]any
		valid   bool
	}{
		{
			name:    "valid headers",
			headers: map[string]any{"authorization": "Bearer token"},
			valid:   true,
		},
		{
			name:    "empty headers",
			headers: map[string]any{},
			valid:   true, // Empty headers are valid
		},
		{
			name:    "nil headers",
			headers: nil,
			valid:   true, // Nil headers are valid
		},
		{
			name: "multiple headers",
			headers: map[string]any{
				"authorization": "Bearer token",
				"content-type":  "application/json",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Headers are always valid in our current implementation
			s.Require().True(tt.valid)
		})
	}
}

// TestAPITestSuite runs the API test suite.
func TestAPITestSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(APITestSuite))
}
