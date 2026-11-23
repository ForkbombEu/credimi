// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/signedtoken"
	"github.com/pocketbase/pocketbase/core"
)

const (
	approvalKeyLabel = "credimi/action-author/approval/v1"
	sessionKeyLabel  = "credimi/action-author/session/v1"
	ticketKeyLabel   = "credimi/action-author/ticket/v1"
	runKeyLabel      = "credimi/action-author/run/v1"
	// ApprovalLifetime bounds how long a signed approval can be used by Save.
	ApprovalLifetime = 30 * time.Minute
)

// Approval is a human's acceptance of one candidate of one ActionAuthor
// workflow, signed by Credimi.
type Approval struct {
	AuthorWorkflowID string `json:"author_workflow_id"`
	CandidateSHA256  string `json:"candidate_sha256"`
	ApproverID       string `json:"approver_id"`
	ApprovedAt       int64  `json:"approved_at"`
	// PlanSHA256 binds the approval to the plan the session saves with.
	PlanSHA256 string `json:"plan_sha256"`
	// Key binds the approval to the candidate key the session saves it as.
	Key string `json:"key"`
	// ProfileSHA256 binds the approval to the FCAF profile the user saw
	// (ProfileSHA256), empty without one.
	ProfileSHA256 string `json:"profile_sha256,omitempty"`
}

// ProfileSHA256 hashes an FCAF profile: the lowercase hex sha256 of its JSON
// encoding (encoding/json sorts map keys), "" for a nil or empty profile or
// one that does not encode, which Save refuses.
func ProfileSHA256(profile map[string]any) string {
	if len(profile) == 0 {
		return ""
	}
	encoded, err := json.Marshal(profile)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

// SignApproval returns the token base64url(json).base64url(hmac).
func SignApproval(app core.App, a Approval) (string, error) {
	return signedtoken.Sign(app, approvalKeyLabel, a)
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
	if err := signedtoken.Verify(app, approvalKeyLabel, token, &approval); err != nil {
		if errors.Is(err, signedtoken.ErrUnsigned) {
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
	// Mode is SessionModeFCAF for an FCAF authoring session, empty for chat.
	Mode string `json:"mode,omitempty"`
	// FCAFSuite is the suite an FCAF session authors, empty for chat.
	FCAFSuite string `json:"fcaf_suite,omitempty"`
}

// SignSession returns the token base64url(json).base64url(hmac) of g.
func SignSession(app core.App, g SessionGrant) (string, error) {
	g.DeviceID = canonify.NormalizePath(g.DeviceID)
	return signedtoken.Sign(app, sessionKeyLabel, g)
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
	return signedtoken.Sign(app, ticketKeyLabel, g)
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
	return signedtoken.Sign(app, runKeyLabel, g)
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
	if err := signedtoken.Verify(app, label, token, &got); err != nil {
		if errors.Is(err, signedtoken.ErrUnsigned) {
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
