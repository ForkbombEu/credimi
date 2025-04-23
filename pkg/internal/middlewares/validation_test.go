// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package middlewares

import (
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/forkbombeu/didimo/pkg/internal/routing"
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
	if !e.nextCalled {
		t.Error("expected Next() to be called")
	}
}

func TestDynamicValidateInputByType_EmptyBody(t *testing.T) {
	handler := DynamicValidateInputByType(reflect.TypeOf(testStruct{}))
	e := &mockRequestEventWithNext{RequestEvent: mockRequestEvent(nil)}
	e.Request.ContentLength = 0
	err := handler((*core.RequestEvent)(e.RequestEvent))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	// if !e.nextCalled {
	// 	t.Error("expected Next() to be called")
	// }
}

func TestDynamicValidateInputByType_InvalidJSON(t *testing.T) {
	handler := DynamicValidateInputByType(reflect.TypeOf(testStruct{}))
	body := strings.NewReader("{invalid json}")
	e := &mockRequestEventWithNext{RequestEvent: mockRequestEvent(body)}
	e.Request.ContentLength = int64(body.Len())
	err := handler((*core.RequestEvent)(e.RequestEvent))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "Invalid JSON format") {
		t.Errorf("expected JSON error, got: %v", err)
	}
}

func TestDynamicValidateInputByType_ValidationFails(t *testing.T) {
	handler := DynamicValidateInputByType(reflect.TypeOf(testStruct{}))
	// Missing required fields
	body := strings.NewReader(`{"name": "", "email": "not-an-email", "age": -5}`)
	e := &mockRequestEventWithNext{RequestEvent: mockRequestEvent(body)}
	e.Request.ContentLength = int64(body.Len())
	err := handler((*core.RequestEvent)(e.RequestEvent))
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "Validation failed") {
		t.Errorf("expected validation error, got: %v", err)
	}
}

func TestDynamicValidateInputByType_ValidationPasses(t *testing.T) {
	handler := DynamicValidateInputByType(reflect.TypeOf(testStruct{}))
	body := strings.NewReader(`{"name": "Alice", "email": "alice@example.com", "age": 30}`)
	e := &mockRequestEventWithNext{RequestEvent: mockRequestEvent(body)}
	e.Request.ContentLength = int64(body.Len())
	err := handler((*core.RequestEvent)(e.RequestEvent))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	// if !e.nextCalled {
	// 	t.Error("expected Next() to be called")
	// }
	val := e.Request.Context().Value(routing.ValidatedInputKey)
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
	err := handler((*core.RequestEvent)(e.RequestEvent))
	if err == nil {
		t.Fatal("expected error for body read failure, got nil")
	}
	if !strings.Contains(err.Error(), "Failed to read request body") {
		t.Errorf("expected body read error, got: %v", err)
	}
}

type errReader struct{}

func (e *errReader) Read(p []byte) (int, error) {
	return 0, errors.New("read error")
}
