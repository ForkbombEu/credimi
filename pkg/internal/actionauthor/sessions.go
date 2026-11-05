// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const sessionsCollection = "action_author_sessions"

var sessionIDPattern = regexp.MustCompile(
	`^ActionAuthorSession-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
)

// IsSessionID reports whether id has the form of the session workflow IDs
// Credimi starts: ActionAuthorSession-<lowercase ticket uuid>.
func IsSessionID(id string) bool {
	return sessionIDPattern.MatchString(id)
}

// SessionDetails are what Credimi lists about a recorded session.
type SessionDetails struct {
	WalletVersionIdentifier string
	DeviceID                string
}

// RecordedSession is a row of action_author_sessions.
type RecordedSession struct {
	WorkflowID, RunID, WalletVersionIdentifier, DeviceID string
	Created                                              time.Time
}

// ErrSessionNotRecorded reports that Credimi has not recorded the run of a
// session workflow (yet: the session's first activity can run before the
// start activity records it).
var ErrSessionNotRecorded = errors.New("action author session is not recorded yet")

// RecordSessionRun records runID as the only run of the session workflow
// that Credimi started in namespace. Recording the same run again is a no-op;
// another run fails with ErrForbidden. details are set only when the row is
// created.
func RecordSessionRun(
	app core.App,
	namespace, workflowID, runID string,
	details SessionDetails,
) error {
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
		record.Set("wallet_version_identifier", details.WalletVersionIdentifier)
		record.Set("device_id", details.DeviceID)
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

// RecentWalletSessions lists the namespace's latest sessions of wallet
// (organization/wallet), newest first: rows whose wallet_version_identifier
// is wallet or starts with wallet + "/".
func RecentWalletSessions(
	app core.App,
	namespace, wallet string,
	limit int,
) ([]RecordedSession, error) {
	records, err := app.FindRecordsByFilter(
		sessionsCollection,
		"namespace = {:ns} && (wallet_version_identifier = {:wallet} || wallet_version_identifier ~ {:prefix})",
		"-created",
		limit,
		0,
		dbx.Params{"ns": namespace, "wallet": wallet, "prefix": wallet + "/%"},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list action author sessions: %w", err)
	}
	sessions := make([]RecordedSession, 0, len(records))
	for _, record := range records {
		sessions = append(sessions, RecordedSession{
			WorkflowID:              record.GetString("workflow_id"),
			RunID:                   record.GetString("run_id"),
			WalletVersionIdentifier: record.GetString("wallet_version_identifier"),
			DeviceID:                record.GetString("device_id"),
			Created:                 record.GetDateTime("created").Time(),
		})
	}
	return sessions, nil
}
