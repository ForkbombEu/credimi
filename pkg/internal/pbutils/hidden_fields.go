// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pbutils

import (
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/list"
)

// LoadHiddenRequestFields loads the submitted values of the given hidden fields
// into the request record and returns them, keyed by field name.
//
// PocketBase drops hidden fields from non-superuser create and update requests.
// Bind it to OnRecordCreateRequest or OnRecordUpdateRequest: those hooks run
// after the collection create or update rule has authorized the write, so the
// caller may set the fields it was already allowed to write. Superuser requests
// keep their hidden fields and return nil.
func LoadHiddenRequestFields(
	e *core.RecordRequestEvent,
	fieldNames ...string,
) (map[string]any, error) {
	if e.HasSuperuserAuth() {
		return nil, nil
	}

	body := map[string]any{}
	if err := e.BindBody(&body); err != nil {
		return nil, err
	}

	submitted := map[string]any{}
	for key, value := range body {
		if slices.Contains(fieldNames, strings.Trim(key, "+-")) {
			submitted[key] = value
		}
	}
	if err := addUploadedFiles(e, body, submitted, fieldNames); err != nil {
		return nil, err
	}

	loaded := map[string]any{}
	for key, value := range e.Record.ReplaceModifiers(submitted) {
		if slices.Contains(fieldNames, key) {
			e.Record.Set(key, value)
			loaded[key] = value
		}
	}
	return loaded, nil
}

// addUploadedFiles merges uploaded files into submitted the way the PocketBase
// record handlers do: plain keys keep the existing file names sent in the body.
func addUploadedFiles(
	e *core.RecordRequestEvent,
	body map[string]any,
	submitted map[string]any,
	fieldNames []string,
) error {
	contentType := e.Request.Header.Get("content-type")
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		return nil
	}

	for _, name := range fieldNames {
		field := e.Collection.Fields.GetByName(name)
		if field == nil || field.Type() != core.FieldTypeFile {
			continue
		}
		for _, key := range []string{name, "+" + name, name + "+"} {
			files, err := e.FindUploadedFiles(key)
			if errors.Is(err, http.ErrMissingFile) {
				continue
			}
			if err != nil {
				return err
			}

			uploaded := make([]any, 0, len(files))
			if key == name && body[key] != nil {
				for _, existing := range list.ToUniqueStringSlice(body[key]) {
					uploaded = append(uploaded, existing)
				}
			}
			for _, file := range files {
				uploaded = append(uploaded, file)
			}
			submitted[key] = uploaded
		}
	}
	return nil
}
