package utils

import (
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// NewValidator func for create a new validator for model fields.
func NewValidator() *validator.Validate {
	// Create a new validator for a Book model.
	validate := validator.New()

	// Custom validation for UUID fields: valid when the value parses as a
	// UUID (a custom func must report true for valid input). uuid.UUID is
	// a [16]byte array, so reflect String() would yield "<uuid.UUID Value>"
	// instead of the ID — extract it via type assertion first; plain
	// strings (e.g. client-supplied IDs) validate from their content.
	_ = validate.RegisterValidation("uuid", func(fl validator.FieldLevel) bool {
		if fl.Field().CanInterface() {
			if id, ok := fl.Field().Interface().(uuid.UUID); ok {
				_, err := uuid.Parse(id.String())
				return err == nil
			}
		}
		_, err := uuid.Parse(fl.Field().String())
		return err == nil
	})

	return validate
}

// ValidatorErrors func for show validation errors for each invalid fields.
func ValidatorErrors(err error) map[string]string {
	// Define fields map.
	fields := map[string]string{}

	// Make error message for each invalid field.
	for _, err := range err.(validator.ValidationErrors) {
		fields[err.Field()] = err.Error()
	}

	return fields
}
