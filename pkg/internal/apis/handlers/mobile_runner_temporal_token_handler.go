// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalauth"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// MobileRunnerTemporalTokenScopeAgent asks for the runner's action author
// token: the same namespace, under the agent's own subject.
const MobileRunnerTemporalTokenScopeAgent = "agent"

// MobileRunnerTemporalTokenRequest names the runner a Temporal token is for.
type MobileRunnerTemporalTokenRequest struct {
	RunnerID string `json:"runner_id" validate:"required"`
	// Scope is empty for the runner's own token, or "agent" for its action
	// author's token, which grants only `runner-<record id>:write`.
	Scope string `json:"scope,omitempty"`
}

// MobileRunnerTemporalTokenResponse carries a runner's Temporal JWT.
type MobileRunnerTemporalTokenResponse struct {
	Token string `json:"token"`
	// ExpiresIn is the token lifetime in seconds.
	ExpiresIn int `json:"expires_in"`
}

// authorizeRunnerKey resolves the runner a runner-facing call names and
// allows it only to owners of the runner's organization (and superusers), only
// to the internal admin key or a superuser for an admin-managed runner, and
// never for a disabled runner. It returns the runner and its canonical ID.
func authorizeRunnerKey(
	e *core.RequestEvent,
	runnerID string,
) (*core.Record, string, *apierror.APIError) {
	return authorizeRunnerManager(
		e.App,
		e.Auth,
		runnerID,
		"only organization owners can obtain a Temporal token",
		"only the internal admin key can obtain a Temporal token for an admin-managed runner",
	)
}

// authorizeRunnerManager resolves runnerID and allows auth to manage the
// runner only when it owns the runner's organization or is a superuser, only
// as a superuser for an admin-managed runner, and never for a disabled runner.
// ownerRequired and adminManaged are the messages of the two role refusals;
// an empty adminManaged lets owners manage admin-managed runners too.
func authorizeRunnerManager(
	app core.App,
	auth *core.Record,
	runnerID, ownerRequired, adminManaged string,
) (*core.Record, string, *apierror.APIError) {
	record, canonicalID, apiErr := resolveLifecycleRunner(app, auth, runnerID)
	if apiErr != nil {
		return nil, "", apiErr
	}
	if !isSuperuserAuth(auth) {
		_, err := app.FindFirstRecordByFilter(
			"orgAuthorizations",
			"user = {:user} && organization = {:org} && role.name = {:role}",
			dbx.Params{"user": auth.Id, "org": record.GetString("owner"), "role": "owner"},
		)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", apierror.New(
				http.StatusForbidden,
				"runner_id",
				"runner_owner_role_required",
				ownerRequired,
			)
		}
		if err != nil {
			return nil, "", apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed_to_check_role",
				err.Error(),
			)
		}
	}
	if adminManaged != "" && record.GetBool("admin_managed") && !isSuperuserAuth(auth) {
		return nil, "", apierror.New(
			http.StatusForbidden,
			"runner_id",
			"admin_managed_runner",
			adminManaged,
		)
	}
	if record.GetBool("disabled") {
		return nil, "", apierror.New(
			http.StatusForbidden,
			"runner_id",
			"runner_disabled",
			"mobile runner is disabled",
		)
	}
	return record, canonicalID, nil
}

// agentNotEnabledError refuses an action author call of a runner whose agent
// is not enabled.
func agentNotEnabledError() *apierror.APIError {
	return apierror.New(
		http.StatusForbidden,
		"scope",
		"agent_not_enabled",
		"the runner's action author is not enabled",
	)
}

// HandleMobileRunnerTemporalToken returns a Temporal JWT that grants the
// runner write access to its runner namespace only. With scope "agent" it
// returns the action author's token instead: the same namespace, under the
// agent's subject.
func HandleMobileRunnerTemporalToken() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[MobileRunnerTemporalTokenRequest](e)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"mobile_runner",
				"invalid_request",
				err.Error(),
			)
		}
		if input.Scope != "" && input.Scope != MobileRunnerTemporalTokenScopeAgent {
			return apierror.New(
				http.StatusBadRequest,
				"scope",
				"invalid_request",
				fmt.Sprintf("unknown token scope %q", input.Scope),
			)
		}

		record, canonicalID, apiErr := authorizeRunnerKey(e, input.RunnerID)
		if apiErr != nil {
			return apiErr
		}

		subject := "mobile-runner:" + canonify.NormalizePath(canonicalID)
		if input.Scope == MobileRunnerTemporalTokenScopeAgent {
			if !record.GetBool("agent_enabled") {
				return agentNotEnabledError()
			}
			subject = "mobile-runner-agent:" + canonify.NormalizePath(canonicalID)
		}
		permissions := mobileRunnerTemporalPermissions(record)

		signer, err := temporalauth.DefaultSigner()
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal_token",
				"failed_to_sign_token",
				err.Error(),
			)
		}
		token, _, err := signer.Sign(subject, permissions, time.Now())
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal_token",
				"failed_to_sign_token",
				err.Error(),
			)
		}

		return e.JSON(http.StatusOK, MobileRunnerTemporalTokenResponse{
			Token:     token,
			ExpiresIn: int(temporalauth.TokenTTL.Seconds()),
		})
	}
}

// mobileRunnerTemporalPermissions returns the permissions of every runner
// token, whatever the runner's kind and the token's scope: write on the
// runner's namespace, prepared or not, and nothing else.
func mobileRunnerTemporalPermissions(record *core.Record) []string {
	return []string{mobilerunner.RunnerNamespace(record.Id) + ":write"}
}
