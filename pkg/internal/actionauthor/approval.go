// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	approvalKeyLabel     = "credimi/action-author/approval/v1"
	approvalMinSecretLen = 32
	// ApprovalLifetime bounds how long a signed approval can be used by Save.
	ApprovalLifetime = 30 * time.Minute
)

var approvalEncoding = base64.RawURLEncoding

// Approval is a human's acceptance of one candidate of one ActionAuthor
// workflow, signed by Credimi.
type Approval struct {
	AuthorWorkflowID   string `json:"author_workflow_id"`
	CandidateSHA256    string `json:"candidate_sha256"`
	ApproverID         string `json:"approver_id"`
	ApproverCollection string `json:"approver_collection"`
	ApprovedAt         int64  `json:"approved_at"`
}

// approvalKey derives the signing key from the users collection's auth token
// secret, so rotating that secret also revokes pending approvals.
func approvalKey(app core.App) ([]byte, error) {
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return nil, fmt.Errorf("find users collection: %w", err)
	}
	secret := users.AuthToken.Secret
	if len(secret) < approvalMinSecretLen {
		return nil, errors.New("users auth token secret is too short to sign approvals")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(approvalKeyLabel))
	return mac.Sum(nil), nil
}

func approvalMAC(key []byte, payload string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(payload))
	return mac.Sum(nil)
}

// SignApproval returns the token base64url(json).base64url(hmac).
func SignApproval(app core.App, a Approval) (string, error) {
	key, err := approvalKey(app)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(a)
	if err != nil {
		return "", fmt.Errorf("encode approval: %w", err)
	}
	payload := approvalEncoding.EncodeToString(encoded)
	return payload + "." + approvalEncoding.EncodeToString(approvalMAC(key, payload)), nil
}

// VerifyApproval accepts only an unexpired token Credimi signed for this
// workflow and candidate; every other token is ErrForbidden.
func VerifyApproval(
	app core.App,
	token, authorWorkflowID, candidateSHA256 string,
	now time.Time,
) (Approval, error) {
	var approval Approval
	forbidden := func(reason string) error {
		return fmt.Errorf("%w: %s", ErrForbidden, reason)
	}
	key, err := approvalKey(app)
	if err != nil {
		return approval, err
	}
	payload, signature, ok := strings.Cut(token, ".")
	if !ok {
		return approval, forbidden("the approval is not signed by Credimi")
	}
	mac, err := approvalEncoding.DecodeString(signature)
	if err != nil || !hmac.Equal(mac, approvalMAC(key, payload)) {
		return approval, forbidden("the approval is not signed by Credimi")
	}
	encoded, err := approvalEncoding.DecodeString(payload)
	if err != nil || json.Unmarshal(encoded, &approval) != nil {
		return Approval{}, forbidden("the approval is not signed by Credimi")
	}
	switch {
	case approval.AuthorWorkflowID != authorWorkflowID:
		return Approval{}, forbidden("the approval is for another session")
	case approval.CandidateSHA256 != candidateSHA256:
		return Approval{}, forbidden("the approval is for another candidate")
	case now.Sub(time.Unix(approval.ApprovedAt, 0)) > ApprovalLifetime:
		return Approval{}, forbidden("the approval expired")
	}
	return approval, nil
}

// OwnsOrganization reports whether the actor holds the owner role of the
// organization, the role Select requires. Superusers own every organization.
func OwnsOrganization(app core.App, in Actor, orgID string) (bool, error) {
	actor, err := findActor(app, in)
	if err != nil {
		return false, err
	}
	if actor.IsSuperuser() {
		return true, nil
	}
	authorizations, err := app.FindRecordsByFilter(
		"orgAuthorizations",
		"user = {:user} && organization = {:org} && role.name = 'owner'",
		"",
		1,
		0,
		dbx.Params{"user": actor.Id, "org": orgID},
	)
	if err != nil {
		return false, fmt.Errorf("check organization owner: %w", err)
	}
	return len(authorizations) > 0, nil
}
