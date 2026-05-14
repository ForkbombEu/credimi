// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"fmt"
	"reflect"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const pipelinesCollectionName = "pipelines"

func RegisterPipelineHooks(app core.App) {
	app.OnRecordUpdate(pipelinesCollectionName).BindFunc(func(e *core.RecordEvent) error {
		persisted, err := e.App.FindRecordById(pipelinesCollectionName, e.Record.Id)
		if err != nil {
			return fmt.Errorf("failed to load existing pipeline record: %w", err)
		}

		if !persisted.GetBool("published") {
			return e.Next()
		}

		if onlyPublicationStatusChanged(persisted, e.Record) {
			return e.Next()
		}

		return apis.NewBadRequestError(
			"Published pipeline records cannot be changed.",
			validation.Errors{
				"pipeline": validation.NewError(
					"validation_pipeline_published_locked",
					"unpublish the pipeline before changing it",
				),
			},
		)
	})
}

func onlyPublicationStatusChanged(original, updated *core.Record) bool {
	for _, fieldName := range original.Collection().Fields.FieldNames() {
		switch fieldName {
		case "published", "created", "updated":
			continue
		}

		if fieldName == "canonified_name" &&
			original.GetString("name") == updated.GetString("name") {
			continue
		}

		if !reflect.DeepEqual(original.Get(fieldName), updated.Get(fieldName)) {
			return false
		}
	}

	return true
}
