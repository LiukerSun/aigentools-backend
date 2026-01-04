package models

import (
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
)

// ParameterDefinition defines the structure of a single parameter in AIModel parameters
type ParameterDefinition struct {
	Name        string      `json:"name" validate:"required"`
	Type        string      `json:"type" validate:"required"`
	Required    *bool       `json:"required" validate:"required"` // Pointer to ensure presence
	Description string      `json:"description" validate:"required"`
	Example     interface{} `json:"example" validate:"required"` // Required, cannot be nil
}

// ModelParameters defines the top-level structure of AIModel parameters
type ModelParameters struct {
	RequestHeader      []ParameterDefinition `json:"request_header" validate:"required,dive"`
	RequestBody        []ParameterDefinition `json:"request_body" validate:"required,dive"`
	ResponseParameters []ParameterDefinition `json:"response_parameters" validate:"required,dive"`
}

// ValidateModelParameters validates the structure of the JSON parameters
// It checks if the parameters conform to the defined schema using validator/v10
func ValidateModelParameters(parameters JSON) error {
	// Convert JSON (map[string]interface{}) to ModelParameters struct
	// First marshal to bytes
	bytes, err := json.Marshal(parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	// Then unmarshal to struct
	var modelParams ModelParameters
	if err := json.Unmarshal(bytes, &modelParams); err != nil {
		// This might happen if the structure is completely different (e.g. types don't match)
		return fmt.Errorf("invalid parameters structure: %w", err)
	}

	// Validate the struct
	validate := validator.New()
	if err := validate.Struct(modelParams); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}
