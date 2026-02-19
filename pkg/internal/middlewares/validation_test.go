// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package middlewares

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

type mockResponseWriter struct {
	header http.Header
	body   bytes.Buffer
	code   int
}

func (m *mockResponseWriter) Header() http.Header         { return m.header }
func (m *mockResponseWriter) Write(b []byte) (int, error) { return m.body.Write(b) }
func (m *mockResponseWriter) WriteHeader(statusCode int)  { m.code = statusCode }

func mockRequestEvent(body io.Reader) *core.RequestEvent {
	req, _ := http.NewRequest("POST", "/", body)
	return &core.RequestEvent{
		Event: router.Event{
			Request:  req,
			Response: &mockResponseWriter{header: make(http.Header)},
		},
	}
}

type testStruct struct {
	Name  string `json:"name"  validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age"   validate:"gte=0,lte=130"`
}

type mockRequestEventWithNext struct {
	*core.RequestEvent
	nextCalled bool
}

func (e *mockRequestEventWithNext) Next() error {
	e.nextCalled = true
	return nil
}

func TestDynamicValidateInputByType_NilInputType(t *testing.T) {
	handler := DynamicValidateInputByType(nil)
	e := &mockRequestEventWithNext{RequestEvent: mockRequestEvent(nil)}
	err := handler(e.RequestEvent)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestDynamicValidateInputByType_EmptyBody(t *testing.T) {
	handler := DynamicValidateInputByType(reflect.TypeOf(testStruct{}))
	e := &mockRequestEventWithNext{RequestEvent: mockRequestEvent(nil)}
	e.Request.ContentLength = 0
	err := handler(e.RequestEvent)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestDynamicValidateInputByType_InvalidJSON(t *testing.T) {
	handler := DynamicValidateInputByType(reflect.TypeOf(testStruct{}))
	body := strings.NewReader("{invalid json}")
	e := &mockRequestEventWithNext{RequestEvent: mockRequestEvent(body)}
	e.Request.ContentLength = int64(body.Len())
	err := handler(e.RequestEvent)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	resp := e.Response.(*mockResponseWriter)
	if resp.code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.code)
	}
	if !strings.Contains(resp.body.String(), "Invalid JSON format") {
		t.Errorf("expected JSON error in body, got: %s", resp.body.String())
	}
}

func TestDynamicValidateInputByType_ValidationFails(t *testing.T) {
	handler := DynamicValidateInputByType(reflect.TypeOf(testStruct{}))
	body := strings.NewReader(`{"name": "", "email": "not-an-email", "age": -5}`)
	e := &mockRequestEventWithNext{RequestEvent: mockRequestEvent(body)}
	e.Request.ContentLength = int64(body.Len())
	err := handler(e.RequestEvent)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	resp := e.Response.(*mockResponseWriter)
	if resp.code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.code)
	}
	if !strings.Contains(resp.body.String(), "Validation failed") {
		t.Errorf("expected validation error in body, got: %s", resp.body.String())
	}
}

func TestDynamicValidateInputByType_ValidationPasses(t *testing.T) {
	handler := DynamicValidateInputByType(reflect.TypeOf(testStruct{}))
	body := strings.NewReader(`{"name": "Alice", "email": "alice@example.com", "age": 30}`)
	e := &mockRequestEventWithNext{RequestEvent: mockRequestEvent(body)}
	e.Request.ContentLength = int64(body.Len())
	err := handler(e.RequestEvent)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	val := e.Request.Context().Value(ValidatedInputKey)
	if val == nil {
		t.Error("expected validated input in context")
	}
	ts, ok := val.(testStruct)
	if !ok {
		t.Errorf("expected context value to be testStruct, got %T", val)
	}
	if ts.Name != "Alice" || ts.Email != "alice@example.com" || ts.Age != 30 {
		t.Errorf("unexpected struct values: %+v", ts)
	}
}

func TestDynamicValidateInputByType_ReadBodyError(t *testing.T) {
	handler := DynamicValidateInputByType(reflect.TypeOf(testStruct{}))
	badReader := &errReader{}
	e := &mockRequestEventWithNext{RequestEvent: mockRequestEvent(badReader)}
	e.Request.ContentLength = 10
	err := handler(e.RequestEvent)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	resp := e.Response.(*mockResponseWriter)
	if resp.code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.code)
	}
	if !strings.Contains(resp.body.String(), "Failed to read request body") {
		t.Errorf("expected body read error in body, got: %s", resp.body.String())
	}
}

type errReader struct{}

func (e *errReader) Read(_ []byte) (int, error) {
	return 0, errors.New("read error")
}

func TestFormatValidationErrors_NonValidationError(t *testing.T) {
	details := formatValidationErrors(errors.New("boom"))
	require.Len(t, details, 1)
	require.Equal(t, "validation process error", details[0]["error"])
	require.Equal(t, "boom", details[0]["message"])
}

func TestFormatValidationErrors_ValidationError(t *testing.T) {
	err := validate.Struct(testStruct{
		Name:  "",
		Email: "not-an-email",
		Age:   -1,
	})
	require.Error(t, err)

	details := formatValidationErrors(err)
	require.GreaterOrEqual(t, len(details), 1)
	for _, detail := range details {
		require.NotEmpty(t, detail["field"])
		require.NotEmpty(t, detail["tag"])
		require.NotNil(t, detail["message"])
	}
}

func TestValidateMapValues(t *testing.T) {
	type mapStructValue struct {
		Field string `validate:"required"`
	}

	t.Run("invalid scalar values", func(t *testing.T) {
		m := map[string]any{"runner": ""}
		err := validateMapValues(reflect.ValueOf(m))
		require.Error(t, err)
		var validationErrs validator.ValidationErrors
		require.ErrorAs(t, err, &validationErrs)
	})

	t.Run("invalid struct and pointer values", func(t *testing.T) {
		empty := &mapStructValue{}
		m := map[string]any{
			"struct":  mapStructValue{},
			"pointer": empty,
		}
		err := validateMapValues(reflect.ValueOf(m))
		require.Error(t, err)
		var validationErrs validator.ValidationErrors
		require.ErrorAs(t, err, &validationErrs)
	})

	t.Run("valid values", func(t *testing.T) {
		v := &mapStructValue{Field: "ok"}
		m := map[string]any{
			"struct": mapStructValue{Field: "ok"},
			"ptr":    v,
			"plain":  "value",
		}
		err := validateMapValues(reflect.ValueOf(m))
		require.NoError(t, err)
	})
}

func TestValidateValue_ScalarAndMap(t *testing.T) {
	t.Run("invalid scalar", func(t *testing.T) {
		errs := validateValue("")
		require.NotEmpty(t, errs)
	})

	t.Run("valid scalar", func(t *testing.T) {
		errs := validateValue("runner-1")
		require.Nil(t, errs)
	})

	t.Run("nil map", func(t *testing.T) {
		var nilMap map[string]any
		errs := validateValue(nilMap)
		require.Nil(t, errs)
	})

	t.Run("invalid map value", func(t *testing.T) {
		errs := validateValue(map[string]any{"runner": ""})
		require.NotEmpty(t, errs)
	})
}
