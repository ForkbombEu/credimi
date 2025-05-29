// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package customchecks

import (
	"github.com/invopop/jsonschema"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func AddHooks(app *pocketbase.PocketBase) {
	app.OnRecordCreateRequest("custom_checks").BindFunc(func(e *core.RecordRequestEvent) error {
		err := generateSchema(e)
		if err != nil {
			return err
		}
		return e.Next()
	})

	app.OnRecordUpdateRequest("custom_checks").BindFunc(func(e *core.RecordRequestEvent) error {
		err := generateSchema(e)
		if err != nil {
			return err
		}
		return e.Next()
	})
}

func generateSchema(e *core.RecordRequestEvent) error {
	inputSample := e.Record.Get("input_json_sample")
	if inputSample == nil {
		return apis.NewBadRequestError("input_sample is required", nil)
	}

	jsonSchema := jsonschema.Reflect(inputSample)
	serializedSchema, err := jsonSchema.MarshalJSON()
	if err != nil {
		return apis.NewInternalServerError("failed to serialize schema", err)
	}

	schemaString := string(serializedSchema)
	e.Record.Set("schema", schemaString)

	return nil
}
