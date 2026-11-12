// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"errors"
	"regexp"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
)

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

// ErrSessionNotVisible reports that Temporal does not list a run of a session
// workflow (yet: its visibility record can lag behind its start).
var ErrSessionNotVisible = errors.New("action author session is not visible in Temporal yet")

// SessionWallet returns the wallet (organization/wallet) of a session's
// wallet_version_identifier: its first two segments.
func SessionWallet(walletVersionIdentifier string) string {
	segments := strings.SplitN(walletVersionIdentifier, "/", 3)
	if len(segments) < 2 {
		return walletVersionIdentifier
	}
	return segments[0] + "/" + segments[1]
}
