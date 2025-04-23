// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package routing

import (
	"fmt"
	"net/http"
	"reflect"
	"github.com/forkbombeu/didimo/pkg/internal/apierror" // Adjust import path
	"github.com/pocketbase/pocketbase/core"
)

type contextKey string

const ValidatedInputKey contextKey = "validatedInput"

type HandlerFunc func(e *core.RequestEvent) error

type RouteDefinition struct {
	Method    string        
	Path      string    
	Handler   HandlerFunc 
	InputType reflect.Type  
}

type InputTypeRegistry map[string]reflect.Type

func BuildRegistryKey(method, path string) string {
	return method + " " + path
}

func GetValidatedInput[T any](e *core.RequestEvent) (T, error) {
	validatedInput := e.Request.Context().Value(ValidatedInputKey)
	var zero T

	if validatedInput == nil {
		return zero, nil
	}
	typedInput, ok := validatedInput.(T)
	if !ok {
		expectedType := fmt.Sprintf("%T", zero)
		actualType := fmt.Sprintf("%T", validatedInput)
		errMsg := fmt.Sprintf("critical type mismatch for validated input: expected %s, got %s", expectedType, actualType)
		return zero, apierror.New(http.StatusInternalServerError, "routing", "Input Type Mismatch", errMsg)
	}
	return typedInput, nil
}

