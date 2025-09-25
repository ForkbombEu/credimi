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

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
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
