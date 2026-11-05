// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const sessionsCollection = "action_author_sessions"

// ErrSessionNotRecorded reports that Credimi has not recorded the run of a
// session workflow (yet: the session's first activity can run before the
// start activity records it).
var ErrSessionNotRecorded = errors.New("action author session is not recorded yet")

// RecordSessionRun records runID as the only run of the session workflow
// that Credimi started in namespace. Recording the same run again is a no-op;
// another run fails with ErrForbidden.
func RecordSessionRun(app core.App, namespace, workflowID, runID string) error {
	return app.RunInTransaction(func(txApp core.App) error {
		recorded, err := SessionRun(txApp, namespace, workflowID)
		switch {
		case err == nil && recorded == runID:
			return nil
		case err == nil:
			return fmt.Errorf(
				"%w: action author session %s already ran as another run",
				ErrForbidden,
				workflowID,
			)
		case !errors.Is(err, ErrSessionNotRecorded):
			return err
		}
		collection, err := txApp.FindCollectionByNameOrId(sessionsCollection)
		if err != nil {
			return fmt.Errorf("failed to find action author sessions: %w", err)
		}
		record := core.NewRecord(collection)
		record.Set("namespace", namespace)
		record.Set("workflow_id", workflowID)
		record.Set("run_id", runID)
		if err := txApp.Save(record); err != nil {
			return fmt.Errorf("failed to record action author session run: %w", err)
		}
		return nil
	})
}

// SessionRun returns the run of the session workflow that Credimi started in
// namespace, or ErrSessionNotRecorded.
func SessionRun(app core.App, namespace, workflowID string) (string, error) {
	record, err := app.FindFirstRecordByFilter(
		sessionsCollection,
		"namespace = {:namespace} && workflow_id = {:workflow}",
		dbx.Params{"namespace": namespace, "workflow": workflowID},
	)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrSessionNotRecorded
	}
	if err != nil {
		return "", fmt.Errorf("failed to find action author session run: %w", err)
	}
	return record.GetString("run_id"), nil
}
