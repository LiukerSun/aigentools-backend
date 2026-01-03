package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ValidationErrorDetail represents the structure of a single validation error.
type ValidationErrorDetail struct {
	Field    string      `json:"field"`
	Message  string      `json:"message"`
	Expected string      `json:"expected"`
	Received interface{} `json:"received"`
}

// ValidationErrorData represents the data field in the validation error response.
type ValidationErrorData struct {
	Errors        []ValidationErrorDetail `json:"errors"`
	Documentation string                  `json:"documentation"`
}

const DocumentationLink = "https://localhost:8080/swagger/index.html" // Replace with actual documentation link

// BindAndValidate binds the request body to the given object and validates it.
// If validation fails, it sends a formatted error response and returns false.
// If validation succeeds, it returns true.
func BindAndValidate(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		var validationErrors []ValidationErrorDetail

		// Handle validator.ValidationErrors
		if errs, ok := err.(validator.ValidationErrors); ok {
			for _, e := range errs {
				// Try to get the actual value using reflection if possible,
				// but ShouldBindJSON doesn't easily expose the raw map before binding.
				// However, validator.FieldError has a Value() method which returns the value
				// that failed validation *if* it was successfully unmarshaled into the struct.
				receivedVal := e.Value()

				// e.Field() returns the struct field name. We might want the JSON tag name.
				// But standard validator uses struct field name by default unless registered with reflect.
				// For simplicity, we'll use the field name or try to find the json tag if we wanted to be very precise.
				// Here we stick to the error field provided by validator.

				detail := ValidationErrorDetail{
					Field:    e.Field(),
					Message:  fmt.Sprintf("Field validation for '%s' failed on the '%s' tag", e.Field(), e.Tag()),
					Expected: e.Param(),
					Received: receivedVal,
				}

				if detail.Expected == "" {
					detail.Expected = e.Tag()
				}

				// Customize messages based on tag
				switch e.Tag() {
				case "required":
					detail.Message = fmt.Sprintf("Field '%s' is required", e.Field())
					detail.Expected = "not null"
				case "email":
					detail.Message = fmt.Sprintf("Field '%s' must be a valid email address", e.Field())
					detail.Expected = "email format"
				case "min":
					detail.Message = fmt.Sprintf("Field '%s' must be at least %s characters long", e.Field(), e.Param())
					detail.Expected = fmt.Sprintf("min length %s", e.Param())
				case "max":
					detail.Message = fmt.Sprintf("Field '%s' must be at most %s characters long", e.Field(), e.Param())
					detail.Expected = fmt.Sprintf("max length %s", e.Param())
				}

				validationErrors = append(validationErrors, detail)
			}
		} else if jsonErr, ok := err.(*json.UnmarshalTypeError); ok {
			// Handle JSON type mismatch errors
			detail := ValidationErrorDetail{
				Field:    jsonErr.Field,
				Message:  fmt.Sprintf("Field '%s' has invalid type", jsonErr.Field),
				Expected: jsonErr.Type.String(),
				Received: jsonErr.Value,
			}
			validationErrors = append(validationErrors, detail)
		} else {
			// Handle other errors (e.g., malformed JSON)
			detail := ValidationErrorDetail{
				Field:    "body",
				Message:  "Malformed JSON or invalid request body",
				Expected: "valid JSON",
				Received: "invalid",
			}
			validationErrors = append(validationErrors, detail)
		}

		response := Response{
			Status:  http.StatusBadRequest,
			Message: "Invalid request parameters",
			Data: ValidationErrorData{
				Errors:        validationErrors,
				Documentation: DocumentationLink,
			},
		}

		c.JSON(http.StatusBadRequest, response)
		return false
	}
	return true
}

// Helper to get json tag name if needed (omitted for now to keep it simple,
// can be added if users complain about case sensitivity)
func getJSONTagName(obj interface{}, fieldName string) string {
	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if f, ok := t.FieldByName(fieldName); ok {
		if tag := f.Tag.Get("json"); tag != "" && tag != "-" {
			return tag
		}
	}
	return fieldName
}
