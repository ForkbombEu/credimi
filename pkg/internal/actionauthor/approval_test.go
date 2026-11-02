// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApprovalSignAndVerify(t *testing.T) {
	app, _, _ := authoringFixture(t)
	now := time.Unix(1_800_000_000, 0)
	sha := strings.Repeat("a", 64)
	approval := Approval{
		AuthorWorkflowID:   "ActionAuthorSession-t-author",
		CandidateSHA256:    sha,
		ApproverID:         "user-1",
		ApproverCollection: "users",
		ApprovedAt:         now.Unix(),
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

func TestOwnsOrganization(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	selection, err := Select(app, namespace, request)
	require.NoError(t, err)
	orgID := selection.Organization.Id

	owns, err := OwnsOrganization(app, request.Actor, orgID)
	require.NoError(t, err)
	assert.True(t, owns)

	other, err := app.FindFirstRecordByData("organizations", "name", "userB's organization")
	require.NoError(t, err)
	owns, err = OwnsOrganization(app, request.Actor, other.Id)
	require.NoError(t, err)
	assert.False(t, owns)

	superuser, err := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, "admin@example.org")
	require.NoError(t, err)
	owns, err = OwnsOrganization(
		app,
		Actor{ActorID: superuser.Id, ActorCollection: core.CollectionNameSuperusers},
		other.Id,
	)
	require.NoError(t, err)
	assert.True(t, owns)

	_, err = OwnsOrganization(app, Actor{ActorID: "missing", ActorCollection: "users"}, orgID)
	require.ErrorIs(t, err, ErrForbidden)
}
