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

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const sessionsCollection = "action_author_sessions"

var sessionIDPattern = regexp.MustCompile(
	`^ActionAuthorSession-([a-z0-9]{15})-([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$`,
)

// IsSessionID reports whether id has the form of the session workflow IDs
// Credimi starts: ActionAuthorSession-<runner record id>-<lowercase ticket uuid>.
func IsSessionID(id string) bool {
	return sessionIDPattern.MatchString(id)
}

// SessionWorkflowID is the workflow ID of the session that ticketID starts on
// a device of runnerID.
func SessionWorkflowID(runnerID, ticketID string) string {
	return "ActionAuthorSession-" + runnerID + "-" + ticketID
}

// SessionRunnerID returns the record ID of the runner sessionID runs on.
func SessionRunnerID(sessionID string) (string, bool) {
	match := sessionIDPattern.FindStringSubmatch(sessionID)
	if match == nil {
		return "", false
	}
	return match[1], true
}

// SessionNamespace returns the Temporal namespace sessionID runs in: its
// runner's agent namespace.
func SessionNamespace(sessionID string) (string, bool) {
	runnerID, ok := SessionRunnerID(sessionID)
	if !ok {
		return "", false
	}
	return canonify.RunnerNamespacePrefix + runnerID, true
}

// SessionDetails are what Credimi lists about a recorded session.
type SessionDetails struct {
	WalletVersionIdentifier string
	DeviceID                string
	// ActorID is the user who started the session.
	ActorID string
	// Organization is the canonified_name of the organization the session
	// was started for; the session itself runs in its runner's namespace.
	Organization string
}

// RecordedSession is a row of action_author_sessions.
type RecordedSession struct {
	WorkflowID, RunID, WalletVersionIdentifier, DeviceID, ActorID string
	// Namespace is the Temporal namespace the session runs in.
	Namespace string
	// Organization is the canonified_name of the session's organization.
	Organization string
	Created      time.Time
}

// ErrSessionNotRecorded reports that Credimi has not recorded the run of a
// session workflow (yet: the session's first activity can run before the
// start activity records it).
var ErrSessionNotRecorded = errors.New("action author session is not recorded yet")

// RecordSessionRun records runID as the only run of the session workflow
// that Credimi started in namespace for details.Organization. Recording the
// same run again is a no-op;
// another run fails with ErrForbidden. details are set only when the row is
// created.
func RecordSessionRun(
	app core.App,
	namespace, workflowID, runID string,
	details SessionDetails,
) error {
	return app.RunInTransaction(func(txApp core.App) error {
		recorded, err := FindSession(txApp, workflowID)
		switch {
		case err == nil && recorded.RunID == runID:
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
		record.Set("actor_id", details.ActorID)
		record.Set("organization", details.Organization)
		if err := txApp.Save(record); err != nil {
			return fmt.Errorf("failed to record action author session run: %w", err)
		}
		return nil
	})
}

// FindSession returns the recorded session workflowID, in any namespace, or
// ErrSessionNotRecorded. Callers check its Namespace or Organization.
func FindSession(app core.App, workflowID string) (RecordedSession, error) {
	record, err := app.FindFirstRecordByFilter(
		sessionsCollection,
		"workflow_id = {:workflow}",
		dbx.Params{"workflow": workflowID},
	)
	if errors.Is(err, sql.ErrNoRows) {
		return RecordedSession{}, ErrSessionNotRecorded
	}
	if err != nil {
		return RecordedSession{}, fmt.Errorf("failed to find action author session run: %w", err)
	}
	return recordedSession(record), nil
}

func recordedSession(record *core.Record) RecordedSession {
	return RecordedSession{
		WorkflowID:              record.GetString("workflow_id"),
		RunID:                   record.GetString("run_id"),
		WalletVersionIdentifier: record.GetString("wallet_version_identifier"),
		DeviceID:                record.GetString("device_id"),
		ActorID:                 record.GetString("actor_id"),
		Namespace:               record.GetString("namespace"),
		Organization:            record.GetString("organization"),
		Created:                 record.GetDateTime("created").Time(),
	}
}

// RecentWalletSessions lists the latest sessions of wallet
// (organization/wallet) that actorID started for organization (its
// canonified_name), newest first: rows whose wallet_version_identifier is
// wallet or starts with wallet + "/".
func RecentWalletSessions(
	app core.App,
	organization, wallet, actorID string,
	limit int,
) ([]RecordedSession, error) {
	records, err := app.FindRecordsByFilter(
		sessionsCollection,
		"organization = {:org} && actor_id = {:actor} && (wallet_version_identifier = {:wallet} || wallet_version_identifier ~ {:prefix})",
		"-created",
		limit,
		0,
		dbx.Params{
			"org":    organization,
			"actor":  actorID,
			"wallet": wallet,
			"prefix": wallet + "/%",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list action author sessions: %w", err)
	}
	sessions := make([]RecordedSession, 0, len(records))
	for _, record := range records {
		sessions = append(sessions, recordedSession(record))
	}
	return sessions, nil
}
