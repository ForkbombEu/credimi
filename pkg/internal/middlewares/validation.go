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
	"reflect"

	"github.com/go-playground/validator/v10"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

var validate = validator.New()

func ValidateInputMiddleware[T any]() func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		raw, err := io.ReadAll(e.Request.Body)
		if err != nil {
			return apis.NewBadRequestError("Invalid JSON body", err)
		}
		ptr := new(T)
		if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&ptr); err != nil {
			return apis.NewBadRequestError("Invalid JSON body i", err)
		}

		tKind := reflect.TypeOf(*ptr).Kind()
		var details []map[string]any

		switch tKind {
		case reflect.Struct:
			if err := validate.Struct(*ptr); err != nil {
				for _, ve := range err.(validator.ValidationErrors) {
					details = append(details, map[string]interface{}{
						"field":   ve.Field(),
						"tag":     ve.Tag(),
						"param":   ve.Param(),
						"value":   ve.Value(),
						"message": ve.Error(),
					})
				}
			}

		case reflect.Map:
			m := reflect.ValueOf(ptr)
			for _, key := range m.MapKeys() {
				val := m.MapIndex(key).Interface()
				vType := reflect.TypeOf(val)

				if vType.Kind() == reflect.Struct || (vType.Kind() == reflect.Ptr && vType.Elem().Kind() == reflect.Struct) {
					if err := validate.Struct(val); err != nil {
						for _, ve := range err.(validator.ValidationErrors) {
							details = append(details, map[string]interface{}{
								"field":   fmt.Sprintf("%v.%s", key, ve.Field()),
								"tag":     ve.Tag(),
								"param":   ve.Param(),
								"value":   ve.Value(),
								"message": ve.Error(),
							})
						}
					}
				} else {
					if err := validate.Var(val, "required"); err != nil {
						details = append(details, map[string]interface{}{
							"field":   fmt.Sprintf("%v", key),
							"message": err.Error(),
						})
					}
				}
			}

		default:
			if err := validate.Var(*ptr, "required"); err != nil {
				details = append(details, map[string]interface{}{
					"field":   "",
					"message": err.Error(),
				})
			}
		}

		if len(details) > 0 {
			return apis.NewBadRequestError("Validation failed", map[string]interface{}{"errors": details})
		}

		e.Request.Body = io.NopCloser(bytes.NewBuffer(raw))


		ctx := context.WithValue(e.Request.Context(), "validatedInput", *ptr)
		e.Request = e.Request.WithContext(ctx)
		return e.Next()
	}
}
