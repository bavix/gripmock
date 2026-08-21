package app

import (
	stderrors "errors"
	"fmt"

	errorFormatter "github.com/bavix/gripmock/v3/internal/infra/errors"
	localstuber "github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// Validation errors.
var (
	ErrServiceIsMissing         = stderrors.New("service name is missing")
	ErrMethodIsMissing          = stderrors.New("method name is missing")
	ErrServiceNotRemovable      = stderrors.New("service not found or not removable")
	ErrEmptyBody                = stderrors.New("empty body")
	ErrFileDescriptorSetNoFiles = stderrors.New("FileDescriptorSet does not contain files")
	ErrResolveDescriptorDeps    = stderrors.New("failed to resolve FileDescriptorSet dependencies")
	ErrInvalidFileDescriptorSet = stderrors.New("invalid FileDescriptorSet")
	ErrRegisterDescriptorFile   = stderrors.New("failed to register descriptor file")

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

func mcpMethodNotFound(message string) error {
	return kindError{kind: ErrMCPToolNotFound, message: message}
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

func mcpUUIDArgError(key, value string, err error) error {
	message := key + " must be a UUID"
	if value != "" {
		message += ": " + value
	}

	if err == nil {
		return mcpInvalidArgError(message)
	}

	return mcpInvalidArgErrorWithCause(message+": "+err.Error(), err)
}

func mcpStringListArgError(key string) error {
	return mcpInvalidArgError(key + " must be a non-empty array of strings")
}

func mcpStubPayloadArgError(err error) error {
	if err == nil {
		return mcpInvalidArgError("invalid stubs payload")
	}

	return mcpInvalidArgErrorWithCause("invalid stubs payload: "+err.Error(), err)
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
