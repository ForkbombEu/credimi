// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package middlewares

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/go-playground/validator/v10"
	"github.com/pocketbase/pocketbase/core"
)

var validate = validator.New()

func formatValidationErrors(err error) []map[string]any {
	var details []map[string]any
	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		// Handle non-validation errors if they can occur
		details = append(details, map[string]any{"error": "validation process error", "message": err.Error()})
		return details
	}
	for _, ve := range validationErrs {
		details = append(details, map[string]any{
			"field":   ve.Namespace(), // Use Namespace for nested fields
			"tag":     ve.Tag(),
			"param":   ve.Param(),
			"value":   fmt.Sprintf("%v", ve.Value()), // Format value safely
			"message": ve.Error(),                    // Default validator message
		})
	}
	return details
}

func validateMapValues(m reflect.Value) error {
	var allErrors validator.ValidationErrors
	for _, key := range m.MapKeys() {
		mapVal := m.MapIndex(key).Interface()

		if vType := reflect.TypeOf(mapVal); vType != nil && (vType.Kind() == reflect.Struct || (vType.Kind() == reflect.Ptr && vType.Elem().Kind() == reflect.Struct)) {
			if err := validate.Struct(mapVal); err != nil {
				if vErrs, ok := err.(validator.ValidationErrors); ok {
					for _, ve := range vErrs {
						// Adjust field namespace if possible/needed, simple prepend here
						// Note: validator doesn't easily support map key prefixes in Namespace out-of-the-box.
						// Custom error formatting might be needed for perfect field paths.
						allErrors = append(allErrors, ve)
					}
				}
			}
		} else {
			if err := validate.Var(mapVal, "required"); err != nil {
				if vErrs, ok := err.(validator.ValidationErrors); ok {
					// Again, namespace/field path might need custom handling for map keys.
					allErrors = append(allErrors, vErrs...)
				}
			}
		}
	}
	if len(allErrors) > 0 {
		return allErrors
	}
	return nil
}

func DynamicValidateInputByType(inputType reflect.Type) func(e *core.RequestEvent) error {
	if inputType == nil {
		return func(e *core.RequestEvent) error {
			return e.Next()
		}
	}

	return func(e *core.RequestEvent) error {
		request := e.Request
		if request.ContentLength == 0 {
			return e.Next()
		}
		raw, err := io.ReadAll(request.Body)
		if err != nil {
			return apierror.New(http.StatusBadRequest, "request.body.read", "Failed to read request body", err.Error())
		}
		request.Body = io.NopCloser(bytes.NewBuffer(raw))

		ptr := reflect.New(inputType).Interface()
		bodyReader := bytes.NewReader(raw)
		decoder := json.NewDecoder(bodyReader)

		if err := decoder.Decode(ptr); err != nil {
			if !(err == io.EOF && len(raw) == 0) {
				return apierror.New(http.StatusBadRequest, "request.body.json", "Invalid JSON format for the expected type", err.Error())
			}
		}
		val := reflect.Indirect(reflect.ValueOf(ptr)).Interface()

		validationErrors := validateValue(val)

		if len(validationErrors) > 0 {
			detailsBytes, _ := json.Marshal(validationErrors)
			return apierror.New(http.StatusBadRequest, "request.validation", "Validation failed", string(detailsBytes))
		}

		ctx := context.WithValue(request.Context(), "validatedInput", val)
		e.Request = request.WithContext(ctx)

		return e.Next()
	}
}

func validateValue(val interface{}) []map[string]any {
	valueToValidate := reflect.ValueOf(val)
	var err error

	switch valueToValidate.Kind() {
	case reflect.Struct:
		err = validate.Struct(val)
	case reflect.Map:
		if valueToValidate.IsValid() && !valueToValidate.IsNil() {
			err = validateMapValues(valueToValidate)
		}
	default:
		err = validate.Var(val, "required") // Example tag

		if err != nil {
			validationErrors := formatValidationErrors(err)
			return validationErrors
		}
	}

	if err != nil {
		return formatValidationErrors(err)
	}
	return nil
}
