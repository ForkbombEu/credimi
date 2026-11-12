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

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
)

const (
	approvalKeyLabel     = "credimi/action-author/approval/v1"
	sessionKeyLabel      = "credimi/action-author/session/v1"
	ticketKeyLabel       = "credimi/action-author/ticket/v1"
	runKeyLabel          = "credimi/action-author/run/v1"
	approvalMinSecretLen = 32
	// ApprovalLifetime bounds how long a signed approval can be used by Save.
	ApprovalLifetime = 30 * time.Minute
)

var tokenEncoding = base64.RawURLEncoding

// Approval is a human's acceptance of one candidate of one ActionAuthor
// workflow, signed by Credimi.
type Approval struct {
	AuthorWorkflowID string `json:"author_workflow_id"`
	CandidateSHA256  string `json:"candidate_sha256"`
	ApproverID       string `json:"approver_id"`
	ApprovedAt       int64  `json:"approved_at"`
	// PlanSHA256 binds the approval to the plan the session saves with.
	PlanSHA256 string `json:"plan_sha256"`
}

// signingKey derives a signing key for one token kind from the users
// collection's auth token secret, so rotating that secret also revokes
// pending approvals.
func signingKey(app core.App, label string) ([]byte, error) {
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return nil, fmt.Errorf("find users collection: %w", err)
	}
	secret := users.AuthToken.Secret
	if len(secret) < approvalMinSecretLen {
		return nil, errors.New("users auth token secret is too short to sign tokens")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(label))
	return mac.Sum(nil), nil
}

func tokenMAC(key []byte, payload string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(payload))
	return mac.Sum(nil)
}

// signToken returns the token base64url(json).base64url(hmac) of v, signed
// with the key of label.
func signToken(app core.App, label string, v any) (string, error) {
	key, err := signingKey(app, label)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("encode token: %w", err)
	}
	payload := tokenEncoding.EncodeToString(encoded)
	return payload + "." + tokenEncoding.EncodeToString(tokenMAC(key, payload)), nil
}

// errUnsigned rejects a token Credimi did not sign with the expected key.
var errUnsigned = errors.New("token not signed by Credimi")

// verifyToken decodes into v a token signed with the key of label. It
// returns errUnsigned for a token that is malformed or signed otherwise, and
// other errors when the key cannot be derived.
func verifyToken(app core.App, label, token string, v any) error {
	key, err := signingKey(app, label)
	if err != nil {
		return err
	}
	payload, signature, ok := strings.Cut(token, ".")
	if !ok {
		return errUnsigned
	}
	mac, err := tokenEncoding.DecodeString(signature)
	if err != nil || !hmac.Equal(mac, tokenMAC(key, payload)) {
		return errUnsigned
	}
	encoded, err := tokenEncoding.DecodeString(payload)
	if err != nil || json.Unmarshal(encoded, v) != nil {
		return errUnsigned
	}
	return nil
}

// SignApproval returns the token base64url(json).base64url(hmac).
func SignApproval(app core.App, a Approval) (string, error) {
	return signToken(app, approvalKeyLabel, a)
}

// VerifyApproval accepts only an unexpired token Credimi signed for this
// workflow and candidate; every other token is ErrForbidden.
func VerifyApproval(
	app core.App,
	token, authorWorkflowID, candidateSHA256 string,
	now time.Time,
) (Approval, error) {
	forbidden := func(reason string) error {
		return fmt.Errorf("%w: %s", ErrForbidden, reason)
	}
	var approval Approval
	if err := verifyToken(app, approvalKeyLabel, token, &approval); err != nil {
		if errors.Is(err, errUnsigned) {
			return Approval{}, forbidden("the approval is not signed by Credimi")
		}
		return Approval{}, err
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

// SessionGrant is Credimi's signature of a session it started after its
// membership, device and allowance checks. Namespace is the Temporal
// namespace the session runs in (its runner's); Organization is the
// canonified_name of the organization it was started for.
type SessionGrant struct {
	SessionWorkflowID       string `json:"session_workflow_id"`
	Namespace               string `json:"namespace"`
	Organization            string `json:"organization"`
	DeviceID                string `json:"device_id"`
	Platform                string `json:"platform"`
	WalletVersionIdentifier string `json:"wallet_version_identifier"`
	ActorID                 string `json:"actor_id"`
}

// SignSession returns the token base64url(json).base64url(hmac) of g.
func SignSession(app core.App, g SessionGrant) (string, error) {
	g.DeviceID = canonify.NormalizePath(g.DeviceID)
	return signToken(app, sessionKeyLabel, g)
}

// VerifySession accepts only a token Credimi signed for exactly want. It has
// no expiry: the grant names the ticket's unique session workflow, and every
// caller also requires that session to be running.
func VerifySession(app core.App, token string, want SessionGrant) error {
	return verifyGrant(
		app,
		sessionKeyLabel,
		token,
		want,
		"action author session was not started by Credimi",
	)
}

// SignTicket returns Credimi's signature of the session g that
// POST /api/action-author/sessions enqueues after its checks. Its key differs
// from SignSession's, so a ticket token copied from the semaphore's history
// is never a session grant.
func SignTicket(app core.App, g SessionGrant) (string, error) {
	g.DeviceID = canonify.NormalizePath(g.DeviceID)
	return signToken(app, ticketKeyLabel, g)
}

// VerifyTicket accepts only a ticket token Credimi signed for exactly want:
// the start activity signs a session only for a ticket Credimi enqueued.
func VerifyTicket(app core.App, token string, want SessionGrant) error {
	return verifyGrant(
		app,
		ticketKeyLabel,
		token,
		want,
		"action author ticket was not enqueued by Credimi",
	)
}

func verifyGrant(app core.App, label, token string, want SessionGrant, refusal string) error {
	want.DeviceID = canonify.NormalizePath(want.DeviceID)
	return verifyExact(app, label, token, want, refusal, func(got *SessionGrant) {
		got.DeviceID = canonify.NormalizePath(got.DeviceID)
	})
}

// RunGrant is Credimi's signature of the run it started for a session: the
// start activity signals it to that run, so a run started later under the
// same ID with a copied start input never carries a valid one.
type RunGrant struct {
	SessionWorkflowID string `json:"session_workflow_id"`
	RunID             string `json:"run_id"`
}

// SignRun returns the token base64url(json).base64url(hmac) of g.
func SignRun(app core.App, g RunGrant) (string, error) {
	return signToken(app, runKeyLabel, g)
}

// VerifyRun accepts only a run token Credimi signed for exactly want.
func VerifyRun(app core.App, token string, want RunGrant) error {
	return verifyExact(
		app,
		runKeyLabel,
		token,
		want,
		"action author session run was not started by Credimi",
	)
}

// verifyExact accepts only a token signed with the key of label whose
// payload, after normalize, is want; every other token is ErrForbidden with
// refusal.
func verifyExact[T comparable](
	app core.App,
	label, token string,
	want T,
	refusal string,
	normalize ...func(*T),
) error {
	var got T
	if err := verifyToken(app, label, token, &got); err != nil {
		if errors.Is(err, errUnsigned) {
			return fmt.Errorf("%w: %s", ErrForbidden, refusal)
		}
		return err
	}
	for _, n := range normalize {
		n(&got)
	}
	if got != want {
		return fmt.Errorf("%w: %s", ErrForbidden, refusal)
	}
	return nil
}
