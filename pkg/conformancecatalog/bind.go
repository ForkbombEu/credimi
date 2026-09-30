// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/pocketbase/dbx"
)

// bindParams builds INSERT bind params from columnSpec + a domain row.
// Each column name must come from overlay or from a matching json tag on row.
// Kind drives array marshaling to JSON text for SQLite.
func bindParams(cols []columnSpec, row any, overlay dbx.Params) (dbx.Params, error) {
	byJSON, err := jsonTaggedFields(row)
	if err != nil {
		return nil, err
	}
	out := make(dbx.Params, len(cols))
	for _, col := range cols {
		if overlay != nil {
			if v, ok := overlay[col.Name]; ok {
				out[col.Name] = v
				continue
			}
		}
		fv, ok := byJSON[col.Name]
		if !ok {
			return nil, fmt.Errorf("bind column %q: no json field and no overlay", col.Name)
		}
		v, err := bindColumnValue(col, fv)
		if err != nil {
			return nil, fmt.Errorf("bind column %q: %w", col.Name, err)
		}
		out[col.Name] = v
	}
	return out, nil
}

func jsonTaggedFields(row any) (map[string]reflect.Value, error) {
	v := reflect.ValueOf(row)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, fmt.Errorf("bind row is nil")
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("bind row must be a struct, got %s", v.Kind())
	}
	t := v.Type()
	out := make(map[string]reflect.Value, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue // unexported
		}
		name := jsonFieldName(f.Tag.Get("json"))
		if name == "" || name == "-" {
			continue
		}
		out[name] = v.Field(i)
	}
	return out, nil
}

func jsonFieldName(tag string) string {
	if tag == "" {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	return name
}

func bindColumnValue(col columnSpec, fv reflect.Value) (any, error) {
	switch col.Kind {
	case ColumnKindStringArray:
		return marshalBindJSONSlice(fv)
	case ColumnKindMemberArray:
		return marshalBindJSONSlice(fv)
	default:
		return fv.Interface(), nil
	}
}

func marshalBindJSONSlice(fv reflect.Value) (string, error) {
	if !fv.IsValid() {
		return "[]", nil
	}
	if fv.Kind() == reflect.Slice && fv.IsNil() {
		return "[]", nil
	}
	b, err := json.Marshal(fv.Interface())
	if err != nil {
		return "", err
	}
	return string(b), nil
}
