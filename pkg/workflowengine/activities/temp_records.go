// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
)

// errTempRecordMismatch reports a temporary record whose owner or identifier
// differs from the expected one.
var errTempRecordMismatch = errors.New("temporary record mismatch")

type DeleteTempRecordInput struct {
	Collection         string `json:"collection"          validate:"required,oneof=credentials use_cases_verifications wallet_versions"`
	RecordID           string `json:"record_id"           validate:"required"`
	ExpectedOwnerID    string `json:"expected_owner_id"   validate:"required"`
	ExpectedIdentifier string `json:"expected_identifier" validate:"required"`
}

type DeleteTempRecordOutput struct {
	Deleted bool `json:"deleted"`
}

// DeleteTempRecordActivity deletes a temporary record created for a CI run.
type DeleteTempRecordActivity struct {
	workflowengine.BaseActivity
	app core.App
}

func NewDeleteTempRecordActivity(app core.App) *DeleteTempRecordActivity {
	return &DeleteTempRecordActivity{
		BaseActivity: workflowengine.BaseActivity{Name: DeleteTempRecordActivityName},
		app:          app,
	}
}

func (a *DeleteTempRecordActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *DeleteTempRecordActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[DeleteTempRecordInput](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	deleted, err := deleteTempRecord(
		a.app,
		payload.Collection,
		payload.RecordID,
		payload.ExpectedOwnerID,
		payload.ExpectedIdentifier,
	)
	if err != nil {
		if errors.Is(err, errTempRecordMismatch) {
			return result, credimiActivityError(
				&a.BaseActivity,
				errorcodes.RecordNotAccessible,
				false,
				err,
			)
		}
		return result, credimiActivityError(
			&a.BaseActivity,
			errorcodes.DatabaseOperationFailed,
			true,
			err,
		)
	}
	result.Output = DeleteTempRecordOutput{Deleted: deleted}
	return result, nil
}

// deleteTempRecord deletes recordID from collection when it belongs to ownerID
// and identifier resolves to it. A missing record returns false and no error.
func deleteTempRecord(
	app core.App,
	collection, recordID, ownerID, identifier string,
) (bool, error) {
	record, err := app.FindRecordById(collection, recordID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("find %s %s: %w", collection, recordID, err)
	}
	if record.GetString("owner") != ownerID {
		return false, fmt.Errorf(
			"%w: owner mismatch: %s %s owner does not match expected_owner_id",
			errTempRecordMismatch,
			collection,
			recordID,
		)
	}
	resolved, err := canonify.Resolve(app, identifier)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("resolve %s: %w", identifier, err)
	}
	if err != nil || resolved.Id != record.Id {
		return false, fmt.Errorf(
			"%w: identifier mismatch: %s does not resolve to %s %s",
			errTempRecordMismatch,
			identifier,
			collection,
			recordID,
		)
	}
	if err := app.Delete(record); err != nil {
		return false, fmt.Errorf("delete %s %s: %w", collection, recordID, err)
	}
	return true, nil
}
