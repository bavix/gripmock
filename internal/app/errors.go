package app

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorFormatter "github.com/bavix/gripmock/v3/internal/infra/errors"
	localstuber "github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// Validation errors.
var (
	ErrInputCannotBeEmpty           = errors.New("input/inputs cannot be empty")
	ErrOutputCannotBeEmpty          = errors.New("output/output.stream cannot be empty")
	ErrInputsCannotBeEmptyForClient = errors.New("inputs cannot be empty for client streaming")
	ErrStreamCannotBeEmptyForServer = errors.New("output.stream cannot be empty for server streaming")
	ErrInputsCannotBeEmptyForBidi   = errors.New("inputs cannot be empty for bidirectional streaming")
	ErrStreamCannotBeEmptyForBidi   = errors.New("output.stream cannot be empty for bidirectional streaming")
	ErrInvalidStubConfiguration     = errors.New("invalid stub configuration")
	ErrInvalidInputConfiguration    = errors.New("cannot have both input and inputs configured")
	ErrInvalidOutputConfiguration   = errors.New("cannot have both output.data and output.stream configured")
)

// ErrorFormatter provides methods for formatting error messages.
type ErrorFormatter struct{}

// NewErrorFormatter creates a new ErrorFormatter instance.
func NewErrorFormatter() *ErrorFormatter {
	return &ErrorFormatter{}
}

// FormatStubNotFoundError formats error messages for stub not found scenarios.
func (f *ErrorFormatter) FormatStubNotFoundError(expect localstuber.Query, result *localstuber.Result) error {
	formatter := errorFormatter.NewStubNotFoundFormatter()

	return formatter.Format(expect, result)
}

// CreateStubNotFoundError creates a gRPC status error for stub not found scenarios.
func (f *ErrorFormatter) CreateStubNotFoundError(serviceName, methodName string, details ...string) error {
	msg := fmt.Sprintf("Failed to find response (service: %s, method: %s)", serviceName, methodName)

	if len(details) > 0 {
		msg += " - " + details[0]
	}

	return status.Error(codes.NotFound, msg)
}

// CreateClientStreamError creates a gRPC status error for client stream scenarios.
func (f *ErrorFormatter) CreateClientStreamError(serviceName, methodName string, err error) error {
	msg := fmt.Sprintf("Failed to find response for client stream (service: %s, method: %s)", serviceName, methodName)

	if err != nil {
		msg += fmt.Sprintf(" - Error: %v", err)
	}

	return status.Error(codes.NotFound, msg)
}
