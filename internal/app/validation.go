package app

import (
	"fmt"
	"sync"

	"github.com/go-playground/validator/v10"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

var (
	stubValidator     *validator.Validate //nolint:gochecknoglobals
	stubValidatorOnce sync.Once           //nolint:gochecknoglobals
)

// NewStubValidator creates a validator with stub-specific custom validations registered.
func NewStubValidator() *validator.Validate {
	v := validator.New()
	if err := v.RegisterValidation("valid_input_config", validateInputConfiguration); err != nil {
		panic("register valid_input_config: " + err.Error())
	}

	if err := v.RegisterValidation("valid_output_config", validateOutputConfiguration); err != nil {
		panic("register valid_output_config: " + err.Error())
	}

	return v
}

func defaultStubValidator() *validator.Validate {
	stubValidatorOnce.Do(func() {
		stubValidator = NewStubValidator()
	})

	return stubValidator
}

// ValidationError represents a validation error with field information.
type ValidationError struct {
	Field   string
	Tag     string
	Value   any
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

// validateInputConfiguration validates that either input or inputs is provided, but not both.
func validateInputConfiguration(fl validator.FieldLevel) bool {
	v := stubFromFieldLevel(fl)
	if v == nil {
		return false
	}

	hasInput := hasValidInputData(v.Input)
	hasInputs := len(v.Inputs) > 0

	// Must have exactly one type of input configuration
	return hasInput != hasInputs
}

// validateOutputConfiguration validates that either data or stream is provided, but not both.
func validateOutputConfiguration(fl validator.FieldLevel) bool {
	v := stubFromFieldLevel(fl)
	if v == nil {
		return false
	}

	hasDataOutput := v.Output.Error != "" || v.Output.Data != nil || v.Output.Code != nil
	hasStreamOutput := len(v.Output.Stream) > 0

	// Must have exactly one type of output configuration
	return hasDataOutput != hasStreamOutput
}

func stubFromFieldLevel(fl validator.FieldLevel) *stuber.Stub {
	// Top() returns the root struct being validated (*Stub when calling Struct(stub))
	if v, ok := fl.Top().Interface().(*stuber.Stub); ok {
		return v
	}

	return nil
}

// Helper functions.
func hasValidInputData(input stuber.InputData) bool {
	return input.Contains != nil || input.Equals != nil || input.Matches != nil
}

func hasValidOutputData(output stuber.Output) bool {
	return output.Error != "" || output.Data != nil || output.Code != nil
}

// getValidationMessage returns a user-friendly validation error message.
func getValidationMessage(fieldError validator.FieldError) string {
	switch fieldError.Tag() {
	case "required":
		return getRequiredFieldMessage(fieldError.Field())
	case "valid_input_config":
		return "Invalid input configuration: must have either 'input' or 'inputs', but not both"
	case "valid_output_config":
		return "Invalid output configuration: must have either 'data' or 'stream', but not both"
	case "gte":
		return "Options.Times must be >= 0 (0 = unlimited matches)"
	case "valid_unary_stub":
		return "Unary stub must have valid input and output"
	case "valid_client_stream_stub":
		return "Client streaming stub must have inputs"
	case "valid_server_stream_stub":
		return "Server streaming stub must have input and stream output"
	case "valid_bidirectional_stub":
		return "Bidirectional streaming stub must have inputs and stream output"
	default:
		return fmt.Sprintf("Validation failed for field %s with tag %s", fieldError.Field(), fieldError.Tag())
	}
}

// getRequiredFieldMessage returns the appropriate error message for required field validation.
func getRequiredFieldMessage(fieldName string) string {
	switch fieldName {
	case "Service":
		return ErrServiceIsMissing.Error()
	case "Method":
		return ErrMethodIsMissing.Error()
	default:
		return fieldName + " is required"
	}
}

// Legacy functions for backward compatibility (deprecated)
// These will be removed in future versions

// hasValidInput checks if the stub has valid input configuration for unary requests.
//
// Deprecated: Use validateStub instead.
func hasValidInput(stub *stuber.Stub) bool {
	return hasValidInputData(stub.Input)
}

// hasValidInputs checks if the stub has valid inputs configuration for client streaming.
//
// Deprecated: Use validateStub instead.
func hasValidInputs(stub *stuber.Stub) bool {
	return len(stub.Inputs) > 0
}

// hasValidOutput checks if the stub has valid output configuration for unary requests.
//
// Deprecated: Use validateStub instead.
func hasValidOutput(stub *stuber.Stub) bool {
	return hasValidOutputData(stub.Output)
}

// hasValidStreamOutput checks if the stub has valid stream output configuration for server streaming.
//
// Deprecated: Use validateStub instead.
func hasValidStreamOutput(stub *stuber.Stub) bool {
	return len(stub.Output.Stream) > 0
}

// validateUnaryStub validates unary stub configuration.
//
// Deprecated: Use validateStub instead.
func validateUnaryStub(stub *stuber.Stub) error {
	if !hasValidInput(stub) {
		return ErrInputCannotBeEmpty
	}

	if !hasValidOutput(stub) {
		return ErrOutputCannotBeEmpty
	}

	return nil
}

// validateClientStreamStub validates client streaming stub configuration.
//
// Deprecated: Use validateStub instead.
func validateClientStreamStub(stub *stuber.Stub) error {
	if !hasValidInputs(stub) {
		return ErrInputsCannotBeEmptyForClient
	}

	return nil
}

// validateServerStreamStub validates server streaming stub configuration.
//
// Deprecated: Use validateStub instead.
func validateServerStreamStub(stub *stuber.Stub) error {
	if !hasValidInput(stub) {
		return ErrInputCannotBeEmpty
	}

	if !hasValidStreamOutput(stub) {
		return ErrStreamCannotBeEmptyForServer
	}

	return nil
}

// validateBidirectionalStub validates bidirectional streaming stub configuration.
//
// Deprecated: Use validateStub instead.
func validateBidirectionalStub(stub *stuber.Stub) error {
	if !hasValidInputs(stub) {
		return ErrInputsCannotBeEmptyForBidi
	}

	if !hasValidStreamOutput(stub) {
		return ErrStreamCannotBeEmptyForBidi
	}

	return nil
}
