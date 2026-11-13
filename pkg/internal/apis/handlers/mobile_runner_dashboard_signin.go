// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// DashboardCallbackPath is the runner dashboard path a sign-in code is sent
// back to.
const DashboardCallbackPath = "/auth/credimi/callback"

// maxDashboardStateLength bounds the dashboard's state, echoed back verbatim.
const maxDashboardStateLength = 128

// dashboardCodeNow is the clock of dashboard sign-in codes; tests replace it.
var dashboardCodeNow = time.Now

// dashboardCodes holds this process's pending dashboard sign-in codes; tests
// replace it.
var dashboardCodes = mobilerunner.NewDashboardCodeStore()

// MobileRunnerDashboardSignInRoutes lets a signed-in user obtain a one-time
// code to sign in to the dashboard of a runner they manage.
var MobileRunnerDashboardSignInRoutes = routing.RouteGroup{
	BaseURL:                "/api/mobile-runner",
	AuthenticationRequired: true,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodPost,
			Path:           "/dashboard-code",
			Handler:        HandleMobileRunnerDashboardCode,
			RequestSchema:  MobileRunnerDashboardCodeRequest{},
			ResponseSchema: MobileRunnerDashboardCodeResponse{},
		},
	},
}

// MobileRunnerDashboardCodeRequest asks for a sign-in code for the dashboard
// of RunnerID, to be sent back to ReturnTo with State.
type MobileRunnerDashboardCodeRequest struct {
	RunnerID string `json:"runner_id" validate:"required"`
	ReturnTo string `json:"return_to" validate:"required"`
	State    string `json:"state"     validate:"required"`
}

// MobileRunnerDashboardCodeResponse carries the dashboard URL the browser goes
// to with the code.
type MobileRunnerDashboardCodeResponse struct {
	Redirect string `json:"redirect"`
}

// MobileRunnerDashboardRedeemRequest redeems a sign-in code for RunnerID,
// sent to the dashboard callback ReturnTo.
type MobileRunnerDashboardRedeemRequest struct {
	RunnerID string `json:"runner_id" validate:"required"`
	Code     string `json:"code"      validate:"required"`
	ReturnTo string `json:"return_to" validate:"required"`
}

// MobileRunnerDashboardRedeemResponse names the user who signed in.
type MobileRunnerDashboardRedeemResponse struct {
	UserID         string `json:"user_id"`
	AuthCollection string `json:"auth_collection"`
	Email          string `json:"email"`
	Name           string `json:"name"`
}

// HandleMobileRunnerDashboardCode issues a one-time code that lets the
// signed-in user into the dashboard of a runner they manage.
func HandleMobileRunnerDashboardCode() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[MobileRunnerDashboardCodeRequest](e)
		if err != nil {
			return dashboardInvalidRequest(err.Error())
		}
		if len(input.State) > maxDashboardStateLength {
			return dashboardInvalidRequest("state is too long")
		}
		if !dashboardBrowserSession(e) {
			return apierror.New(
				http.StatusForbidden,
				"auth",
				"browser_session_required",
				"sign in to Credimi to open a runner dashboard",
			)
		}
		redirect, apiErr := dashboardRedirect(input.ReturnTo)
		if apiErr != nil {
			return apiErr
		}
		record, _, apiErr := authorizeDashboardUser(e.App, e.Auth, input.RunnerID)
		if apiErr != nil {
			return apiErr
		}

		now := dashboardCodeNow()
		code, err := dashboardCodes.Issue(mobilerunner.DashboardCode{
			RunnerID:       record.Id,
			UserID:         e.Auth.Id,
			AuthCollection: e.Auth.Collection().Name,
			ReturnTo:       input.ReturnTo,
			Expires:        now.Add(mobilerunner.DashboardCodeTTL),
		}, now)
		if errors.Is(err, mobilerunner.ErrTooManyUserDashboardCodes) {
			return apierror.New(
				http.StatusTooManyRequests,
				"dashboard_code",
				"too_many_codes",
				"too many pending sign-ins; try again in a minute",
			)
		}
		if errors.Is(err, mobilerunner.ErrTooManyDashboardCodes) {
			return apierror.New(
				http.StatusServiceUnavailable,
				"dashboard_code",
				"too_many_codes",
				"too many pending dashboard sign-ins, try again in a minute",
			)
		}
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"dashboard_code",
				"failed_to_issue_code",
				err.Error(),
			)
		}

		query := redirect.Query()
		query.Set("code", code)
		query.Set("state", input.State)
		redirect.RawQuery = query.Encode()
		return e.JSON(http.StatusOK, MobileRunnerDashboardCodeResponse{
			Redirect: redirect.String(),
		})
	}
}

// HandleMobileRunnerDashboardRedeem redeems a dashboard sign-in code for the
// runner whose key calls it and names the user who signed in.
func HandleMobileRunnerDashboardRedeem() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[MobileRunnerDashboardRedeemRequest](e)
		if err != nil {
			return dashboardInvalidRequest(err.Error())
		}
		record, _, apiErr := authorizeDashboardRunnerKey(e, input.RunnerID)
		if apiErr != nil {
			return apiErr
		}

		// A code sent to another dashboard is refused like a missing one, so
		// the answer reveals nothing about it.
		code, ok := dashboardCodes.Redeem(input.Code, dashboardCodeNow())
		if !ok || code.RunnerID != record.Id || code.ReturnTo != input.ReturnTo {
			return apierror.New(
				http.StatusBadRequest,
				"code",
				"invalid_code",
				"the sign-in code is invalid, expired or already used: sign in again",
			)
		}

		user, err := e.App.FindRecordById(code.AuthCollection, code.UserID)
		if err != nil {
			return apierror.New(
				http.StatusForbidden,
				"code",
				"user_not_found",
				"the user who signed in no longer exists",
			)
		}
		// The user's role may have changed since the code was issued.
		if _, _, apiErr := authorizeDashboardUser(e.App, user, input.RunnerID); apiErr != nil {
			return apiErr
		}

		return e.JSON(http.StatusOK, MobileRunnerDashboardRedeemResponse{
			UserID:         user.Id,
			AuthCollection: user.Collection().Name,
			Email:          user.Email(),
			Name:           user.GetString("name"),
		})
	}
}

// authorizeDashboardUser allows user to sign in to the dashboard of the
// runner runnerID when they may manage it (authorizeRunnerManager).
func authorizeDashboardUser(
	app core.App,
	user *core.Record,
	runnerID string,
) (*core.Record, string, *apierror.APIError) {
	return authorizeRunnerManager(
		app,
		user,
		runnerID,
		"only owners of the runner's organization can sign in to its dashboard",
		"only superusers can sign in to the dashboard of an admin-managed runner",
	)
}

// authorizeDashboardRunnerKey checks the key the runner's dashboard calls with
// as authorizeRunnerKey does, with refusals that name the dashboard.
func authorizeDashboardRunnerKey(
	e *core.RequestEvent,
	runnerID string,
) (*core.Record, string, *apierror.APIError) {
	return authorizeRunnerManager(
		e.App,
		e.Auth,
		runnerID,
		"only keys of the runner's organization owners can serve its dashboard sign-in",
		"only the internal admin key can serve the dashboard sign-in of an admin-managed runner",
	)
}

// dashboardBrowserSession reports whether the caller signed in to Credimi
// with a user or superuser token, not with an API key: a leaked key must not
// open dashboards.
func dashboardBrowserSession(e *core.RequestEvent) bool {
	if strings.TrimSpace(e.Request.Header.Get("Authorization")) == "" ||
		e.Auth == nil || e.Auth.Collection() == nil {
		return false
	}
	name := e.Auth.Collection().Name
	return name == "users" || name == core.CollectionNameSuperusers
}

// dashboardRedirect accepts returnTo only as an absolute http(s) URL of a
// dashboard sign-in callback: no credentials, query or fragment, and the
// escaped path exactly DashboardCallbackPath (no percent-encoded variant).
func dashboardRedirect(returnTo string) (*url.URL, *apierror.APIError) {
	invalid := apierror.New(
		http.StatusBadRequest,
		"return_to",
		"invalid_return_to",
		"return_to must be the http(s) URL of a runner dashboard's "+DashboardCallbackPath,
	)
	u, err := url.Parse(returnTo)
	if err != nil ||
		(u.Scheme != "http" && u.Scheme != "https") ||
		u.Host == "" ||
		u.User != nil ||
		u.Opaque != "" ||
		u.RawPath != "" ||
		u.EscapedPath() != DashboardCallbackPath ||
		u.RawQuery != "" || u.ForceQuery ||
		u.Fragment != "" || strings.Contains(returnTo, "#") {
		return nil, invalid
	}
	return u, nil
}

func dashboardInvalidRequest(message string) *apierror.APIError {
	return apierror.New(http.StatusBadRequest, "mobile_runner", "invalid_request", message)
}
