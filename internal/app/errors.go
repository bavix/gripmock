package app

import (
	stderrors "errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorFormatter "github.com/bavix/gripmock/v3/internal/infra/errors"
	localstuber "github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// Validation errors.
var (
	ErrInputCannotBeEmpty           = stderrors.New("input/inputs cannot be empty")
	ErrOutputCannotBeEmpty          = stderrors.New("output/output.stream cannot be empty")
	ErrInputsCannotBeEmptyForClient = stderrors.New("inputs cannot be empty for client streaming")
	ErrStreamCannotBeEmptyForServer = stderrors.New("output.stream cannot be empty for server streaming")
	ErrInputsCannotBeEmptyForBidi   = stderrors.New("inputs cannot be empty for bidirectional streaming")
	ErrStreamCannotBeEmptyForBidi   = stderrors.New("output.stream cannot be empty for bidirectional streaming")
	ErrInvalidStubConfiguration     = stderrors.New("invalid stub configuration")
	ErrInvalidInputConfiguration    = stderrors.New("cannot have both input and inputs configured")
	ErrInvalidOutputConfiguration   = stderrors.New("cannot have both output.data and output.stream configured")
	ErrServiceIsMissing             = stderrors.New("service name is missing")
	ErrMethodIsMissing              = stderrors.New("method name is missing")
	ErrServiceNotRemovable          = stderrors.New("service not found or not removable")
	ErrEmptyBody                    = stderrors.New("empty body")
	ErrFileDescriptorSetNoFiles     = stderrors.New("FileDescriptorSet does not contain files")
	ErrResolveDescriptorDeps        = stderrors.New("failed to resolve FileDescriptorSet dependencies")
	ErrInvalidFileDescriptorSet     = stderrors.New("invalid FileDescriptorSet")
	ErrRegisterDescriptorFile       = stderrors.New("failed to register descriptor file")

	ErrMCPInvalidRequest  = stderrors.New("mcp invalid request")
	ErrMCPInvalidArgument = stderrors.New("mcp invalid argument")
	ErrMCPToolNotFound    = stderrors.New("mcp tool not found")
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

type kindError struct {
	kind    error
	cause   error
	message string
}

func (e kindError) Error() string {
	return e.message
}

func (e kindError) Unwrap() []error {
	if e.cause == nil {
		return []error{e.kind}
	}

	return []error{e.kind, e.cause}
}

func mcpInvalidArgError(message string) error {
	return kindError{kind: ErrMCPInvalidArgument, message: message}
}

func mcpInvalidArgErrorWithCause(message string, cause error) error {
	return kindError{kind: ErrMCPInvalidArgument, cause: cause, message: message}
}

func mcpInvalidRequestError() error {
	return kindError{kind: ErrMCPInvalidRequest, message: "invalid JSON-RPC request"}
}

func mcpMethodNotFound(message string) error {
	return kindError{kind: ErrMCPToolNotFound, message: message}
}

func mcpRPCMethodNotFoundError() error {
	return mcpMethodNotFound("method not found")
}

func mcpUnknownTool(name string) error {
	return mcpMethodNotFound("unknown tool: " + name)
}

func mcpNonNegativeIntegerArgError(key string) error {
	return mcpInvalidArgError(key + " must be a non-negative integer")
}

func mcpRequiredArgError(key string) error {
	return mcpInvalidArgError(key + " is required")
}

func mcpDescriptorSetBase64ArgError(err error) error {
	if err == nil {
		return mcpInvalidArgError("invalid descriptorSetBase64")
	}

	return mcpInvalidArgErrorWithCause("invalid descriptorSetBase64: "+err.Error(), err)
}

func mcpDescriptorRegistrationArgError(err error) error {
	if err == nil {
		return mcpInvalidArgError("invalid descriptor registration")
	}

	return mcpInvalidArgErrorWithCause(err.Error(), err)
}

func invalidFileDescriptorSetError(err error) error {
	message := ErrInvalidFileDescriptorSet.Error()
	if err != nil {
		message += ": " + err.Error()
	}

	return kindError{kind: ErrInvalidFileDescriptorSet, cause: err, message: message}
}

func registerDescriptorFileError(fileName string, err error) error {
	message := "failed to register file " + fileName
	if err == nil {
		return kindError{kind: ErrRegisterDescriptorFile, message: message}
	}

	return kindError{kind: ErrRegisterDescriptorFile, cause: err, message: message + ": " + err.Error()}
}

func serviceNotRemovable(serviceID string) error {
	message := fmt.Sprintf("service %s not found or not removable", serviceID)

	return kindError{kind: ErrServiceNotRemovable, message: message}
}
