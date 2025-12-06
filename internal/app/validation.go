package app

import (
	"fmt"

	"github.com/go-playground/validator/v10"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

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

// validationStub is a struct used for validation with tags.
type validationStub struct {
	Service string             `json:"service" validate:"required"`
	Method  string             `json:"method"  validate:"required"`
	Input   stuber.InputData   `json:"input"   validate:"valid_input_config"`
	Inputs  []stuber.InputData `json:"inputs"  validate:"valid_input_config"`
	Output  stuber.Output      `json:"output"  validate:"valid_output_config"`
}

// validateInputConfiguration validates that either input or inputs is provided, but not both.
func validateInputConfiguration(fl validator.FieldLevel) bool {
	v, ok := fl.Parent().Interface().(validationStub)
	if !ok {
		return false
	}

	hasInput := hasValidInputData(v.Input)
	hasInputs := len(v.Inputs) > 0

	// Must have exactly one type of input configuration
	return hasInput != hasInputs
}

// validateOutputConfiguration validates that either data or stream is provided, but not both.
func validateOutputConfiguration(fl validator.FieldLevel) bool {
	v, ok := fl.Parent().Interface().(validationStub)
	if !ok {
		return false
	}

	hasDataOutput := v.Output.Error != "" || v.Output.Data != nil || v.Output.Code != nil
	hasStreamOutput := len(v.Output.Stream) > 0

	// Must have exactly one type of output configuration
	return hasDataOutput != hasStreamOutput
}

// Helper functions.
func hasValidInputData(input stuber.InputData) bool {
	return input.Contains != nil || input.Equals != nil || input.Matches != nil
}

func hasValidOutputData(output stuber.Output) bool {
	return output.Error != "" || output.Data != nil || output.Code != nil
}

// validateStubType validates stub based on its type (unary, client stream, server stream, bidirectional).
func validateStubType(stub *stuber.Stub) error {
	// Determine stub type based on input/output configuration
	hasInput := hasValidInputData(stub.Input)
	hasInputs := len(stub.Inputs) > 0
	hasDataOutput := hasValidOutputData(stub.Output)
	hasStreamOutput := len(stub.Output.Stream) > 0

	// Determine stub type
	switch {
	case hasInput && hasDataOutput:
		// Unary stub
		return nil
	case hasInputs && hasDataOutput:
		// Client streaming stub
		return nil
	case hasInput && hasStreamOutput:
		// Server streaming stub
		return nil
	case hasInputs && hasStreamOutput:
		// Bidirectional streaming stub
		return nil
	default:
		// If we can't determine the type, return an error
		return &ValidationError{
			Field:   "Stub",
			Tag:     "invalid_type",
			Value:   stub,
			Message: "Invalid stub configuration: unable to determine stub type",
		}
	}
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
