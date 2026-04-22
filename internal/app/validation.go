package app

import (
	"fmt"

	"github.com/go-playground/validator/v10"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
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

	hasDataOutput := v.Output.Error != "" || v.Output.Data != nil || v.Output.Code != nil || len(v.Output.Details) > 0
	hasStreamOutput := len(v.Output.Stream) > 0

	return hasDataOutput != hasStreamOutput
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
	if input.Contains != nil || input.Equals != nil || input.Matches != nil {
		return true
	}

	for _, alt := range input.AnyOf {
		if alt.Contains != nil || alt.Equals != nil || alt.Matches != nil {
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
		return "Invalid output configuration: must have either 'data' or 'stream', but not both"
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
