// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package middlewares

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func mockRequestEvent(body io.Reader) *core.RequestEvent {
	req, _ := http.NewRequest("POST", "/", body)
	return &core.RequestEvent{
		Event: router.Event{
			Request: req,
		},
	}
}

type testStruct struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"gte=0,lte=130"`
}

func TestValidateInputMiddleware_Struct_Valid(t *testing.T) {
	input := testStruct{Name: "Alice", Email: "alice@example.com", Age: 30}
	b, _ := json.Marshal(input)
	e := mockRequestEvent(bytes.NewReader(b))

	mw := ValidateInputMiddleware[testStruct]()
	err := mw(e)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	val := e.Request.Context().Value("validatedInput")
	ts, ok := val.(testStruct)
	if !ok || ts != input {
		t.Fatalf("validatedInput not set correctly, got %#v", val)
	}
}

func TestValidateInputMiddleware_Struct_Invalid(t *testing.T) {
	input := testStruct{Name: "", Email: "not-an-email", Age: 200}
	b, _ := json.Marshal(input)
	e := mockRequestEvent(bytes.NewReader(b))

	mw := ValidateInputMiddleware[testStruct]()
	err := mw(e)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*router.ApiError)
	if !ok {
		t.Fatalf("expected ApiError, got %T", err)
	}
	if apiErr.Status != 400 {
		t.Errorf("expected 400 error code, got %d", apiErr.Status)
	}
}

func TestValidateInputMiddleware_InvalidJSON(t *testing.T) {
	e := mockRequestEvent(bytes.NewReader([]byte("{invalid json")))
	mw := ValidateInputMiddleware[testStruct]()
	err := mw(e)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*router.ApiError)
	if !ok {
		t.Fatalf("expected ApiError, got %T", err)
	}
	if apiErr.Status != 400 {
		t.Errorf("expected 400 error code, got %d", apiErr.Status)
	}
}

func TestValidateInputMiddleware_Map_Valid(t *testing.T) {
	input := map[string]interface{}{
		"foo": "bar",
		"baz": 123,
	}
	b, _ := json.Marshal(input)
	e := mockRequestEvent(bytes.NewReader(b))

	mw := ValidateInputMiddleware[map[string]interface{}]()
	err := mw(e)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateInputMiddleware_Map_Invalid(t *testing.T) {
	input := map[string]any{
		"foo": "",
		"baz": nil,
	}
	b, _ := json.Marshal(input)
	e := mockRequestEvent(bytes.NewReader(b))

	mw := ValidateInputMiddleware[map[string]any]()
	err := mw(e)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*router.ApiError)
	if !ok {
		t.Fatalf("expected ApiError, got %T", err)
	}
	if apiErr.Status != 400 {
		t.Errorf("expected 400 error code, got %d", apiErr.Status)
	}
}

func TestValidateInputMiddleware_Scalar_Valid(t *testing.T) {
	e := mockRequestEvent(bytes.NewReader([]byte(`"hello"`)))
	mw := ValidateInputMiddleware[string]()
	err := mw(e)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateInputMiddleware_Scalar_Invalid(t *testing.T) {
	e := mockRequestEvent(bytes.NewReader([]byte(`""`)))
	mw := ValidateInputMiddleware[string]()
	err := mw(e)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*router.ApiError)
	if !ok {
		t.Fatalf("expected ApiError, got %T", err)
	}
	if apiErr.Status != 400 {
		t.Errorf("expected 400 error code, got %d", apiErr.Status)
	}
}
