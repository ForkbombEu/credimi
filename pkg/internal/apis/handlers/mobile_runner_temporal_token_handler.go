// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalauth"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

var errReservedNamespace = errors.New("organization namespace is reserved")

// MobileRunnerTemporalTokenRequest names the runner a Temporal token is for.
type MobileRunnerTemporalTokenRequest struct {
	RunnerID string `json:"runner_id" validate:"required"`
}

// MobileRunnerTemporalTokenResponse carries a runner's Temporal JWT.
type MobileRunnerTemporalTokenResponse struct {
	Token string `json:"token"`
	// ExpiresIn is the token lifetime in seconds.
	ExpiresIn int `json:"expires_in"`
}

// HandleMobileRunnerTemporalToken returns a Temporal JWT that grants the
// runner write access to the namespaces its workers poll, plus its agent
// namespace when its action author is enabled.
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

		record, canonicalID, apiErr := resolveLifecycleRunner(e.App, e.Auth, input.RunnerID)
		if apiErr != nil {
			return apiErr
		}
		if !isSuperuserAuth(e.Auth) {
			_, err := e.App.FindFirstRecordByFilter(
				"orgAuthorizations",
				"user = {:user} && organization = {:org} && role.name = {:role}",
				dbx.Params{"user": e.Auth.Id, "org": record.GetString("owner"), "role": "owner"},
			)
			if errors.Is(err, sql.ErrNoRows) {
				return apierror.New(
					http.StatusForbidden,
					"runner_id",
					"runner_owner_role_required",
					"only organization owners can obtain a Temporal token",
				)
			}
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"organization",
					"failed_to_check_role",
					err.Error(),
				)
			}
		}
		if record.GetBool("admin_managed") && !isSuperuserAuth(e.Auth) {
			return apierror.New(
				http.StatusForbidden,
				"runner_id",
				"admin_managed_runner",
				"only the internal admin key can obtain a Temporal token for an admin-managed runner",
			)
		}
		if record.GetBool("disabled") {
			return apierror.New(
				http.StatusForbidden,
				"runner_id",
				"runner_disabled",
				"mobile runner is disabled",
			)
		}

		namespaces, err := mobileRunnerTemporalNamespaces(e.App, record)
		if err != nil {
			if errors.Is(err, errReservedNamespace) {
				return apierror.New(
					http.StatusConflict,
					"organizations",
					"organization_namespace_reserved",
					"rename the organization: its namespace is reserved",
				)
			}
			return apierror.New(
				http.StatusInternalServerError,
				"organizations",
				"failed_to_list_namespaces",
				err.Error(),
			)
		}
		// An enabled agent polls the runner's own namespace, ready or not.
		if record.GetBool("agent_enabled") {
			namespaces = append(namespaces, mobilerunner.AgentNamespace(record.Id))
			slices.Sort(namespaces)
			namespaces = slices.Compact(namespaces)
		}
		permissions := make([]string, len(namespaces))
		for i, ns := range namespaces {
			permissions[i] = ns + ":write"
		}

		signer, err := temporalauth.DefaultSigner()
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal_token",
				"failed_to_sign_token",
				err.Error(),
			)
		}
		token, _, err := signer.Sign(
			"mobile-runner:"+canonify.NormalizePath(canonicalID),
			permissions,
			time.Now(),
		)
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

// mobileRunnerTemporalNamespaces returns the sorted, deduplicated namespaces a
// runner's workers may poll: admin-managed runners serve default and every
// organization, published tenant runners their own and every published
// organization, other tenant runners their own organization only.
func mobileRunnerTemporalNamespaces(app core.App, runner *core.Record) ([]string, error) {
	if runner.GetBool("admin_managed") {
		orgs, err := allOrganizationNamespaces(app)
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, "default")
		slices.Sort(orgs)
		return slices.Compact(orgs), nil
	}
	if runner.GetBool("published") {
		namespaces, err := visibleOrganizationNamespaces(app, runner.GetString("owner"))
		if err != nil {
			return nil, err
		}
		return nonEmptyNamespaces(withoutReservedNamespaces(namespaces))
	}
	org, err := app.FindRecordById("organizations", runner.GetString("owner"))
	if err != nil {
		return nil, fmt.Errorf("find runner owner organization: %w", err)
	}
	namespace := org.GetString("canonified_name")
	if namespace == "" {
		return nil, fmt.Errorf("runner owner organization has no namespace")
	}
	return nonEmptyNamespaces(withoutReservedNamespaces([]string{namespace}))
}

// withoutReservedNamespaces drops namespaces canonify.IsReservedNamespace rejects, keeping order.
func withoutReservedNamespaces(namespaces []string) []string {
	return slices.DeleteFunc(slices.Clone(namespaces), canonify.IsReservedNamespace)
}

// nonEmptyNamespaces fails with errReservedNamespace when a tenant runner has
// no namespace left after dropping reserved ones.
func nonEmptyNamespaces(namespaces []string) ([]string, error) {
	if len(namespaces) == 0 {
		return nil, errReservedNamespace
	}
	return namespaces, nil
}
