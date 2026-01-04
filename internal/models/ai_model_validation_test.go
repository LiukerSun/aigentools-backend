package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateModelParameters(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "Valid parameters",
			input: `{
				"request_header": [
					{"name": "Authorization", "type": "string", "required": true, "description": "Bearer token", "example": "Bearer 123"}
				],
				"request_body": [
					{"name": "prompt", "type": "string", "required": true, "description": "User prompt", "example": "Hello"}
				],
				"response_parameters": [
					{"name": "content", "type": "string", "required": true, "description": "AI response", "example": "Hi there"}
				]
			}`,
			wantErr: false,
		},
		{
			name: "Valid parameters empty arrays",
			input: `{
				"request_header": [],
				"request_body": [],
				"response_parameters": []
			}`,
			wantErr: false,
		},
		{
			name: "Missing request_header",
			input: `{
				"request_body": [],
				"response_parameters": []
			}`,
			wantErr: true,
		},
		{
			name: "Missing request_body",
			input: `{
				"request_header": [],
				"response_parameters": []
			}`,
			wantErr: true,
		},
		{
			name: "Missing response_parameters",
			input: `{
				"request_header": [],
				"request_body": []
			}`,
			wantErr: true,
		},
		{
			name: "Missing name in parameter",
			input: `{
				"request_header": [],
				"request_body": [
					{"type": "string", "required": true, "description": "User prompt", "example": "Hello"}
				],
				"response_parameters": []
			}`,
			wantErr: true,
		},
		{
			name: "Missing required field in parameter",
			input: `{
				"request_header": [],
				"request_body": [
					{"name": "prompt", "type": "string", "description": "User prompt", "example": "Hello"}
				],
				"response_parameters": []
			}`,
			wantErr: true,
		},
		{
			name: "Missing example in parameter",
			input: `{
				"request_header": [],
				"request_body": [
					{"name": "prompt", "type": "string", "required": true, "description": "User prompt"}
				],
				"response_parameters": []
			}`,
			wantErr: true,
		},
		{
			name: "Example is null",
			input: `{
				"request_header": [],
				"request_body": [
					{"name": "prompt", "type": "string", "required": true, "description": "User prompt", "example": null}
				],
				"response_parameters": []
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var params JSON
			err := json.Unmarshal([]byte(tt.input), &params)
			assert.NoError(t, err, "Failed to unmarshal test input")

			err = ValidateModelParameters(params)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
