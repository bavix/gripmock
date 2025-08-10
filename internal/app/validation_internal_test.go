package app

import (
	"testing"

	"github.com/gripmock/stuber"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestHasValidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		stub     *stuber.Stub
		expected bool
	}{
		{
			name: "valid input with Contains",
			stub: &stuber.Stub{
				Input: stuber.InputData{
					Contains: map[string]any{"key": "value"},
				},
			},
			expected: true,
		},
		{
			name: "valid input with Equals",
			stub: &stuber.Stub{
				Input: stuber.InputData{
					Equals: map[string]any{"key": "value"},
				},
			},
			expected: true,
		},
		{
			name: "valid input with Matches",
			stub: &stuber.Stub{
				Input: stuber.InputData{
					Matches: map[string]any{"key": "value"},
				},
			},
			expected: true,
		},
		{
			name: "invalid input - all nil",
			stub: &stuber.Stub{
				Input: stuber.InputData{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := hasValidInput(tt.stub)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestHasValidInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		stub     *stuber.Stub
		expected bool
	}{
		{
			name: "valid inputs - non-empty",
			stub: &stuber.Stub{
				Inputs: []stuber.InputData{
					{Contains: map[string]any{"key": "value"}},
				},
			},
			expected: true,
		},
		{
			name:     "invalid inputs - empty",
			stub:     &stuber.Stub{Inputs: []stuber.InputData{}},
			expected: false,
		},
		{
			name:     "invalid inputs - nil",
			stub:     &stuber.Stub{Inputs: nil},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := hasValidInputs(tt.stub)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestHasValidOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		stub     *stuber.Stub
		expected bool
	}{
		{
			name: "valid output with Error",
			stub: &stuber.Stub{
				Output: stuber.Output{
					Error: "some error",
				},
			},
			expected: true,
		},
		{
			name: "valid output with Data",
			stub: &stuber.Stub{
				Output: stuber.Output{
					Data: map[string]any{"key": "value"},
				},
			},
			expected: true,
		},
		{
			name: "valid output with Code",
			stub: &stuber.Stub{
				Output: stuber.Output{
					Code: func() *codes.Code {
						code := codes.NotFound

						return &code
					}(),
				},
			},
			expected: true,
		},
		{
			name: "invalid output - all empty",
			stub: &stuber.Stub{
				Output: stuber.Output{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := hasValidOutput(tt.stub)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestHasValidStreamOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		stub     *stuber.Stub
		expected bool
	}{
		{
			name: "valid stream output - non-empty",
			stub: &stuber.Stub{
				Output: stuber.Output{
					Stream: []any{map[string]any{"key": "value"}},
				},
			},
			expected: true,
		},
		{
			name: "invalid stream output - empty",
			stub: &stuber.Stub{
				Output: stuber.Output{
					Stream: []any{},
				},
			},
			expected: false,
		},
		{
			name: "invalid stream output - nil",
			stub: &stuber.Stub{
				Output: stuber.Output{
					Stream: nil,
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := hasValidStreamOutput(tt.stub)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateUnaryStub(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		stub        *stuber.Stub
		expectError bool
		errorType   error
	}{
		{
			name: "valid unary stub",
			stub: &stuber.Stub{
				Input: stuber.InputData{
					Contains: map[string]any{"key": "value"},
				},
				Output: stuber.Output{
					Data: map[string]any{"result": "success"},
				},
			},
			expectError: false,
		},
		{
			name: "invalid unary stub - no input",
			stub: &stuber.Stub{
				Input: stuber.InputData{},
				Output: stuber.Output{
					Data: map[string]any{"result": "success"},
				},
			},
			expectError: true,
			errorType:   ErrInputCannotBeEmpty,
		},
		{
			name: "invalid unary stub - no output",
			stub: &stuber.Stub{
				Input: stuber.InputData{
					Contains: map[string]any{"key": "value"},
				},
				Output: stuber.Output{},
			},
			expectError: true,
			errorType:   ErrOutputCannotBeEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateUnaryStub(tt.stub)
			if tt.expectError {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.errorType)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateClientStreamStub(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		stub        *stuber.Stub
		expectError bool
		errorType   error
	}{
		{
			name: "valid client stream stub",
			stub: &stuber.Stub{
				Inputs: []stuber.InputData{
					{Contains: map[string]any{"key": "value"}},
				},
			},
			expectError: false,
		},
		{
			name:        "invalid client stream stub - no inputs",
			stub:        &stuber.Stub{Inputs: []stuber.InputData{}},
			expectError: true,
			errorType:   ErrInputsCannotBeEmptyForClient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateClientStreamStub(tt.stub)
			if tt.expectError {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.errorType)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateServerStreamStub(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		stub        *stuber.Stub
		expectError bool
		errorType   error
	}{
		{
			name: "valid server stream stub",
			stub: &stuber.Stub{
				Input: stuber.InputData{
					Contains: map[string]any{"key": "value"},
				},
				Output: stuber.Output{
					Stream: []any{map[string]any{"key": "value"}},
				},
			},
			expectError: false,
		},
		{
			name: "invalid server stream stub - no input",
			stub: &stuber.Stub{
				Input: stuber.InputData{},
				Output: stuber.Output{
					Stream: []any{map[string]any{"key": "value"}},
				},
			},
			expectError: true,
			errorType:   ErrInputCannotBeEmpty,
		},
		{
			name: "invalid server stream stub - no stream output",
			stub: &stuber.Stub{
				Input: stuber.InputData{
					Contains: map[string]any{"key": "value"},
				},
				Output: stuber.Output{
					Stream: []any{},
				},
			},
			expectError: true,
			errorType:   ErrStreamCannotBeEmptyForServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateServerStreamStub(tt.stub)
			if tt.expectError {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.errorType)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateBidirectionalStub(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		stub        *stuber.Stub
		expectError bool
		errorType   error
	}{
		{
			name: "valid bidirectional stub",
			stub: &stuber.Stub{
				Inputs: []stuber.InputData{
					{Contains: map[string]any{"key": "value"}},
				},
				Output: stuber.Output{
					Stream: []any{map[string]any{"key": "value"}},
				},
			},
			expectError: false,
		},
		{
			name: "invalid bidirectional stub - no inputs",
			stub: &stuber.Stub{
				Inputs: []stuber.InputData{},
				Output: stuber.Output{
					Stream: []any{map[string]any{"key": "value"}},
				},
			},
			expectError: true,
			errorType:   ErrInputsCannotBeEmptyForBidi,
		},
		{
			name: "invalid bidirectional stub - no stream output",
			stub: &stuber.Stub{
				Inputs: []stuber.InputData{
					{Contains: map[string]any{"key": "value"}},
				},
				Output: stuber.Output{
					Stream: []any{},
				},
			},
			expectError: true,
			errorType:   ErrStreamCannotBeEmptyForBidi,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateBidirectionalStub(tt.stub)
			if tt.expectError {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.errorType)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

//nolint:funlen
func TestValidateStub_WithValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		stub    *stuber.Stub
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid unary stub",
			stub: &stuber.Stub{
				Service: "TestService",
				Method:  "TestMethod",
				Input: stuber.InputData{
					Equals: map[string]any{"key": "value"},
				},
				Output: stuber.Output{
					Data: map[string]any{"result": "success"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid client streaming stub",
			stub: &stuber.Stub{
				Service: "TestService",
				Method:  "TestMethod",
				Inputs: []stuber.InputData{
					{Equals: map[string]any{"key": "value"}},
				},
				Output: stuber.Output{
					Data: map[string]any{"result": "success"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid server streaming stub",
			stub: &stuber.Stub{
				Service: "TestService",
				Method:  "TestMethod",
				Input: stuber.InputData{
					Equals: map[string]any{"key": "value"},
				},
				Output: stuber.Output{
					Stream: []any{"message1", "message2"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid bidirectional streaming stub",
			stub: &stuber.Stub{
				Service: "TestService",
				Method:  "TestMethod",
				Inputs: []stuber.InputData{
					{Equals: map[string]any{"key": "value"}},
				},
				Output: stuber.Output{
					Stream: []any{"message1", "message2"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing service",
			stub: &stuber.Stub{
				Method: "TestMethod",
				Input: stuber.InputData{
					Equals: map[string]any{"key": "value"},
				},
				Output: stuber.Output{
					Data: map[string]any{"result": "success"},
				},
			},
			wantErr: true,
			errMsg:  "service name is missing",
		},
		{
			name: "missing method",
			stub: &stuber.Stub{
				Service: "TestService",
				Input: stuber.InputData{
					Equals: map[string]any{"key": "value"},
				},
				Output: stuber.Output{
					Data: map[string]any{"result": "success"},
				},
			},
			wantErr: true,
			errMsg:  "method name is missing",
		},
		{
			name: "invalid input configuration - both input and inputs",
			stub: &stuber.Stub{
				Service: "TestService",
				Method:  "TestMethod",
				Input: stuber.InputData{
					Equals: map[string]any{"key": "value"},
				},
				Inputs: []stuber.InputData{
					{Equals: map[string]any{"key2": "value2"}},
				},
				Output: stuber.Output{
					Data: map[string]any{"result": "success"},
				},
			},
			wantErr: true,
			errMsg:  "Invalid input configuration: must have either 'input' or 'inputs', but not both",
		},
		{
			name: "invalid output configuration - both data and stream",
			stub: &stuber.Stub{
				Service: "TestService",
				Method:  "TestMethod",
				Input: stuber.InputData{
					Equals: map[string]any{"key": "value"},
				},
				Output: stuber.Output{
					Data:   map[string]any{"result": "success"},
					Stream: []any{"message1"},
				},
			},
			wantErr: true,
			errMsg:  "Invalid output configuration: must have either 'data' or 'stream', but not both",
		},
		{
			name: "unary stub without input",
			stub: &stuber.Stub{
				Service: "TestService",
				Method:  "TestMethod",
				Output: stuber.Output{
					Data: map[string]any{"result": "success"},
				},
			},
			wantErr: true,
			errMsg:  "Invalid input configuration: must have either 'input' or 'inputs', but not both",
		},
		{
			name: "unary stub without output",
			stub: &stuber.Stub{
				Service: "TestService",
				Method:  "TestMethod",
				Input: stuber.InputData{
					Equals: map[string]any{"key": "value"},
				},
				Output: stuber.Output{},
			},
			wantErr: true,
			errMsg:  "Invalid output configuration: must have either 'data' or 'stream', but not both",
		},
		{
			name: "client streaming stub without inputs",
			stub: &stuber.Stub{
				Service: "TestService",
				Method:  "TestMethod",
				Output: stuber.Output{
					Data: map[string]any{"result": "success"},
				},
			},
			wantErr: true,
			errMsg:  "Invalid input configuration: must have either 'input' or 'inputs', but not both",
		},
		{
			name: "server streaming stub without input",
			stub: &stuber.Stub{
				Service: "TestService",
				Method:  "TestMethod",
				Output: stuber.Output{
					Stream: []any{"message1"},
				},
			},
			wantErr: true,
			errMsg:  "Invalid input configuration: must have either 'input' or 'inputs', but not both",
		},
		{
			name: "server streaming stub without stream output",
			stub: &stuber.Stub{
				Service: "TestService",
				Method:  "TestMethod",
				Input: stuber.InputData{
					Equals: map[string]any{"key": "value"},
				},
				Output: stuber.Output{
					Data: map[string]any{"result": "success"},
				},
			},
			wantErr: false, // This is actually a valid unary stub
		},
		{
			name: "bidirectional streaming stub without inputs",
			stub: &stuber.Stub{
				Service: "TestService",
				Method:  "TestMethod",
				Output: stuber.Output{
					Stream: []any{"message1"},
				},
			},
			wantErr: true,
			errMsg:  "Invalid input configuration: must have either 'input' or 'inputs', but not both",
		},
		{
			name: "bidirectional streaming stub without stream output",
			stub: &stuber.Stub{
				Service: "TestService",
				Method:  "TestMethod",
				Inputs: []stuber.InputData{
					{Equals: map[string]any{"key": "value"}},
				},
				Output: stuber.Output{
					Data: map[string]any{"result": "success"},
				},
			},
			wantErr: false, // This is actually a valid client streaming stub
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateStub(tt.stub)

			if tt.wantErr {
				require.Error(t, err)

				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	t.Parallel()

	validationErr := &ValidationError{
		Field:   "Service",
		Tag:     "required",
		Value:   "",
		Message: "Service is required",
	}

	require.Equal(t, "Service is required", validationErr.Error())
}
