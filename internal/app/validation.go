package app

import (
	stderrors "errors"
	"fmt"

	"github.com/go-playground/validator/v10"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

func NewStubValidator() (*validator.Validate, error) {
	v := validator.New()

	for name, fn := range map[string]validator.Func{
		"valid_input_config":  validateInputConfiguration,
		"valid_output_config": validateOutputConfiguration,
		"valid_effects":       validateEffectsConfiguration,
	} {
		if err := v.RegisterValidation(name, fn); err != nil {
			return nil, fmt.Errorf("register validation %q: %w", name, err)
		}
	}

	return v, nil
}

func mustNewStubValidator() *validator.Validate {
	v, err := NewStubValidator()
	if err != nil {
		panic("stub validator init: " + err.Error())
	}

	return v
}

type ValidationError struct {
	Field   string
	Tag     string
	Value   any
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

func StubValidator(v *validator.Validate, engine *template.Engine) func(*stuber.Stub) error {
	return func(stub *stuber.Stub) error {
		return checkStub(v, engine, stub)
	}
}

func checkStub(v *validator.Validate, engine *template.Engine, stub *stuber.Stub) error {
	if err := v.Struct(stub); err != nil {
		validationErrors, ok := stderrors.AsType[validator.ValidationErrors](err)
		if !ok || len(validationErrors) == 0 {
			return err
		}

		fieldError := validationErrors[0]

		return &ValidationError{
			Field:   fieldError.Field(),
			Tag:     fieldError.Tag(),
			Value:   fieldError.Value(),
			Message: getValidationMessage(fieldError),
		}
	}

	if !stub.Output.HasTemplate() {
		return nil
	}

	if stub.Service == HealthServiceFullName {
		return templateError(errHealthTemplate)
	}

	document, _ := stub.Output.Document()
	if document == "" {
		return templateError("template output requires data or stream to hold the template text")
	}

	if err := engine.Validate(document); err != nil {
		return templateError("Invalid output template: " + err.Error())
	}

	return nil
}

func templateError(message string) error {
	return &ValidationError{
		Field:   "Template",
		Tag:     "valid_template",
		Value:   true,
		Message: message,
	}
}

func validateInputConfiguration(fl validator.FieldLevel) bool {
	v := stubFromFieldLevel(fl)
	if v == nil {
		return false
	}

	hasInput := hasValidInputData(v.Input)
	hasInputs := len(v.Inputs) > 0

	return hasInput != hasInputs
}

func validateOutputConfiguration(fl validator.FieldLevel) bool {
	v := stubFromFieldLevel(fl)
	if v == nil {
		return false
	}

	return isValidOutputConfiguration(v.Output)
}

func isValidOutputConfiguration(output stuber.Output) bool {
	if output.Data != nil && output.Stream != nil {
		return false
	}

	if output.HasTemplate() {
		document, _ := output.Document()

		return document != ""
	}

	if _, isText := output.Stream.(string); isText {
		return false
	}

	return output.Data != nil || output.Stream != nil ||
		output.Error != "" || output.Code != nil || len(output.Details) > 0
}

func validateEffectsConfiguration(fl validator.FieldLevel) bool {
	v := stubFromFieldLevel(fl)
	if v == nil {
		return false
	}

	for _, effect := range v.Effects {
		switch effect.Action {
		case stuber.EffectActionUpsert:
			if len(effect.Stub) == 0 {
				return false
			}
		case stuber.EffectActionDelete:
			if effect.ID == "" {
				return false
			}
		default:
			return false
		}
	}

	return true
}

func stubFromFieldLevel(fl validator.FieldLevel) *stuber.Stub {
	if v, ok := fl.Top().Interface().(*stuber.Stub); ok {
		return v
	}

	return nil
}

func hasValidInputData(input stuber.InputData) bool {
	if input.Contains != nil || input.Equals != nil || input.Matches != nil || input.Glob != nil {
		return true
	}

	for _, alt := range input.AnyOf {
		if alt.Contains != nil || alt.Equals != nil || alt.Matches != nil || alt.Glob != nil {
			return true
		}
	}

	return false
}

func getValidationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return requiredFieldMessage(fe.Field())
	case "valid_input_config":
		return "Invalid input configuration: must have either 'input' or 'inputs', but not both"
	case "valid_output_config":
		return "Invalid output configuration: 'data' and 'stream' are mutually exclusive, " +
			"and 'template: true' requires the chosen one to hold the template text"
	case "valid_effects":
		return "Invalid effects configuration: upsert requires 'stub', delete requires 'id'"
	case "gte":
		return "Options.Times must be >= 0 (0 = unlimited matches)"
	default:
		return fmt.Sprintf("Validation failed for field %s with tag %s", fe.Field(), fe.Tag())
	}
}

func requiredFieldMessage(field string) string {
	switch field {
	case "Service":
		return ErrServiceIsMissing.Error()
	case "Method":
		return ErrMethodIsMissing.Error()
	default:
		return field + " is required"
	}
}
