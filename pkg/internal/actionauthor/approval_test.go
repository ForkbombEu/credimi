// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApprovalSignAndVerify(t *testing.T) {
	app, _, _ := authoringFixture(t)
	now := time.Unix(1_800_000_000, 0)
	sha := strings.Repeat("a", 64)
	approval := Approval{
		AuthorWorkflowID: "ActionAuthorSession-t-author",
		CandidateSHA256:  sha,
		ApproverID:       "user-1",
		ApprovedAt:       now.Unix(),
	}
	token, err := SignApproval(app, approval)
	require.NoError(t, err)

	verified, err := VerifyApproval(
		app,
		token,
		approval.AuthorWorkflowID,
		sha,
		now.Add(29*time.Minute),
	)
	require.NoError(t, err)
	assert.Equal(t, approval, verified)

	payload, signature, _ := strings.Cut(token, ".")
	other, err := SignApproval(app, Approval{
		AuthorWorkflowID: approval.AuthorWorkflowID,
		CandidateSHA256:  sha,
		ApproverID:       "user-2",
		ApprovedAt:       now.Unix(),
	})
	require.NoError(t, err)
	otherPayload, _, _ := strings.Cut(other, ".")
	for name, tc := range map[string]struct {
		token, workflowID, sha string
		at                     time.Time
		message                string
	}{
		"forged":           {token: "forged.x", message: "not signed by Credimi"},
		"unsigned":         {token: payload, message: "not signed by Credimi"},
		"tampered payload": {token: otherPayload + "." + signature, message: "not signed by Credimi"},
		"wrong workflow":   {workflowID: "ActionAuthorSession-u-author", message: "another session"},
		"wrong candidate":  {sha: strings.Repeat("b", 64), message: "another candidate"},
		"expired":          {at: now.Add(31 * time.Minute), message: "the approval expired"},
	} {
		t.Run(name, func(t *testing.T) {
			if tc.token == "" {
				tc.token = token
			}
			if tc.workflowID == "" {
				tc.workflowID = approval.AuthorWorkflowID
			}
			if tc.sha == "" {
				tc.sha = sha
			}
			if tc.at.IsZero() {
				tc.at = now
			}
			_, err := VerifyApproval(app, tc.token, tc.workflowID, tc.sha, tc.at)
			require.ErrorIs(t, err, ErrForbidden)
			assert.ErrorContains(t, err, tc.message)
		})
	}

	// Rotating the users token secret revokes pending approvals.
	users, err := app.FindCollectionByNameOrId("users")
	require.NoError(t, err)
	users.AuthToken.Secret = strings.Repeat("r", 50)
	require.NoError(t, app.Save(users))
	_, err = VerifyApproval(app, token, approval.AuthorWorkflowID, sha, now)
	require.ErrorIs(t, err, ErrForbidden)

	users.AuthToken.Secret = "short"
	require.NoError(t, app.SaveNoValidate(users))
	_, err = SignApproval(app, approval)
	require.ErrorContains(t, err, "too short")
}

func TestSessionSignAndVerify(t *testing.T) {
	app, _, _ := authoringFixture(t)
	grant := SessionGrant{
		SessionWorkflowID:       "ActionAuthorSession-ticket",
		Namespace:               "org-a",
		DeviceID:                "org-a/runner/pixel",
		Platform:                "android",
		WalletVersionIdentifier: "org-a/wallet/v1",
		ActorID:                 "user-1",
	}
	token, err := SignSession(app, grant)
	require.NoError(t, err)
	require.NoError(t, VerifySession(app, token, grant))
	// Device paths compare normalized.
	withSlash := grant
	withSlash.DeviceID = "/" + grant.DeviceID
	require.NoError(t, VerifySession(app, token, withSlash))

	payload, _, _ := strings.Cut(token, ".")
	changed := map[string]func(*SessionGrant){
		"session":   func(g *SessionGrant) { g.SessionWorkflowID = "ActionAuthorSession-other" },
		"namespace": func(g *SessionGrant) { g.Namespace = "org-b" },
		"device":    func(g *SessionGrant) { g.DeviceID = "org-a/runner/other" },
		"platform":  func(g *SessionGrant) { g.Platform = "ios" },
		"wallet":    func(g *SessionGrant) { g.WalletVersionIdentifier = "org-a/wallet/v2" },
		"actor":     func(g *SessionGrant) { g.ActorID = "user-2" },
	}
	for name, change := range changed {
		t.Run(name, func(t *testing.T) {
			want := grant
			change(&want)
			err := VerifySession(app, token, want)
			require.ErrorIs(t, err, ErrForbidden)
			assert.ErrorContains(t, err, "action author session was not started by Credimi")
		})
	}
	for name, bad := range map[string]string{"unsigned": payload, "empty": ""} {
		t.Run(name, func(t *testing.T) {
			err := VerifySession(app, bad, grant)
			require.ErrorIs(t, err, ErrForbidden)
			assert.ErrorContains(t, err, "action author session was not started by Credimi")
		})
	}
}

func TestTicketAndSessionTokensAreNotInterchangeable(t *testing.T) {
	app, _, _ := authoringFixture(t)
	grant := SessionGrant{
		SessionWorkflowID:       "ActionAuthorSession-ticket",
		Namespace:               "org-a",
		DeviceID:                "org-a/runner/pixel",
		Platform:                "android",
		WalletVersionIdentifier: "org-a/wallet/v1",
		ActorID:                 "user-1",
	}
	ticket, err := SignTicket(app, grant)
	require.NoError(t, err)
	require.NoError(t, VerifyTicket(app, ticket, grant))
	other := grant
	other.ActorID = "user-2"
	err = VerifyTicket(app, ticket, other)
	require.ErrorIs(t, err, ErrForbidden)
	assert.ErrorContains(t, err, "action author ticket was not enqueued by Credimi")

	// A ticket copied from the semaphore's history is no session grant.
	err = VerifySession(app, ticket, grant)
	require.ErrorIs(t, err, ErrForbidden)
	assert.ErrorContains(t, err, "action author session was not started by Credimi")
	session, err := SignSession(app, grant)
	require.NoError(t, err)
	err = VerifyTicket(app, session, grant)
	require.ErrorIs(t, err, ErrForbidden)
}
