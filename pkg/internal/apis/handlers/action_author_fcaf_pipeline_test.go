// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/walletroles"
	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const (
	fcafPipelineURL         = "/api/action-author/fcaf-pipeline"
	fcafPipelineDevice      = authoringAdminRunner + "/android-emulator"
	fcafPipelineHaipVP      = "fcaf-wallet-solution-relying-party-engagement-haip-vp.yaml"
	fcafPipelinePrivateName = "private-runner"
)

// stubFCAFPipelineQueue makes runners reachable and records the enqueued
// pipeline tickets.
func stubFCAFPipelineQueue(t *testing.T) *[]workflows.MobileDeviceSemaphoreEnqueueRunRequest {
	t.Helper()
	origEnqueue := enqueueRunTicket
	origReachable := checkRunnerReachable
	t.Cleanup(func() {
		enqueueRunTicket = origEnqueue
		checkRunnerReachable = origReachable
	})
	checkRunnerReachable = func(context.Context, *core.Record) (bool, error) { return true, nil }
	enqueued := &[]workflows.MobileDeviceSemaphoreEnqueueRunRequest{}
	enqueueRunTicket = func(
		_ context.Context,
		_ string,
		req workflows.MobileDeviceSemaphoreEnqueueRunRequest,
	) (workflows.MobileDeviceSemaphoreEnqueueRunResponse, error) {
		*enqueued = append(*enqueued, req)
		return workflows.MobileDeviceSemaphoreEnqueueRunResponse{
			TicketID: req.TicketID,
			Status:   workflowengine.MobileDeviceSemaphoreRunQueued,
			LineLen:  1,
		}, nil
	}
	return enqueued
}

// setupFCAFPipelineApp is setupActionAuthorApp with a private runner of
// userB's organization and, saved as current FCAF actions of userA's
// wallet, every role of the suite but those in without.
func setupFCAFPipelineApp(t testing.TB, without ...string) *tests.TestApp {
	t.Helper()
	app := setupActionAuthorApp(t)
	foreignOrg, err := app.FindFirstRecordByData("organizations", "name", "userB's organization")
	require.NoError(t, err)
	createAuthoringDevice(
		t,
		app,
		createAuthoringRunner(t, app, foreignOrg.Id, fcafPipelinePrivateName, false),
		"android_emulator",
	)
	if without == nil {
		return app
	}
	scope, err := activities.LoadFCAFScope(actionauthor.FCAFRelyingPartySuite)
	require.NoError(t, err)
	wallet, err := app.FindFirstRecordByData("wallets", "name", "Authoring handler wallet")
	require.NoError(t, err)
	actions, err := app.FindCollectionByNameOrId("wallet_actions")
	require.NoError(t, err)
	if actions.Fields.GetByName("authoring_provenance") == nil {
		actions.Fields.Add(&core.JSONField{Name: "authoring_provenance", MaxSize: 65536})
		require.NoError(t, app.Save(actions))
	}
	for _, role := range scope.Contract.Roles {
		if slices.Contains(without, role.Name) {
			continue
		}
		action := core.NewRecord(actions)
		action.Set("owner", wallet.GetString("owner"))
		action.Set("wallet", wallet.Id)
		action.Set("name", role.Name)
		action.Set("code", "appId: eu.wallet\n---\n- launchApp\n")
		action.Set("category", role.Category)
		action.Set("authoring_provenance", actionauthor.Provenance{
			FCAFRole:       role.Name,
			FCAFRoleSHA256: role.SHA256(),
		})
		require.NoError(t, app.Save(action))
	}
	return app
}

func fcafPipelineBody(device string, scenarios ...string) string {
	quoted := make([]string, len(scenarios))
	for i, scenario := range scenarios {
		quoted[i] = `"` + scenario + `"`
	}
	return `{"wallet_version_identifier":"` + authoringAndroidVersion +
		`","device_id":"` + device + `","scenarios":[` + strings.Join(quoted, ",") + `]}`
}

func TestActionAuthorFCAFPipelineErrors(t *testing.T) {
	foreignDevice := authoringForeignNS + "/" + fcafPipelinePrivateName + "/android-emulator"
	for _, tc := range []struct {
		name, user, body, reason string
		status                   int
	}{
		{
			name:   "wallet the caller cannot author",
			user:   "userB",
			body:   fcafPipelineBody(authoringForeignRunner + "/android-emulator"),
			status: http.StatusForbidden,
			reason: "forbidden",
		},
		{
			name:   "no written role",
			user:   "userA",
			body:   fcafPipelineBody(fcafPipelineDevice),
			status: http.StatusUnprocessableEntity,
			reason: "fcaf_nothing_to_run",
		},
		{
			name:   "unknown scenario",
			user:   "userA",
			body:   fcafPipelineBody(fcafPipelineDevice, fcafPipelineHaipVP, "nope.yaml"),
			status: http.StatusUnprocessableEntity,
			reason: "fcaf_unknown_scenario",
		},
		{
			name:   "private device of another organization",
			user:   "userA",
			body:   fcafPipelineBody(foreignDevice),
			status: http.StatusForbidden,
			reason: "device_id is not accessible",
		},
		{
			name:   "unknown device",
			user:   "userA",
			body:   fcafPipelineBody(authoringAdminRunner + "/missing"),
			status: http.StatusNotFound,
			reason: "device not found",
		},
		{
			name:   "iOS simulator",
			user:   "userA",
			body:   fcafPipelineBody(authoringAdminRunner + "/ios-simulator"),
			status: http.StatusUnprocessableEntity,
			reason: "fcaf_not_supported",
		},
	} {
		(&tests.ApiScenario{
			Name:   tc.name,
			Method: http.MethodPost,
			URL:    fcafPipelineURL,
			Body:   strings.NewReader(tc.body),
			Headers: map[string]string{
				"Authorization": "Bearer " + authoringToken(t, tc.user),
				"Content-Type":  "application/json",
			},
			ExpectedStatus:  tc.status,
			ExpectedContent: []string{`"reason":"` + tc.reason + `"`},
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return setupFCAFPipelineApp(t) },
		}).Test(t)
	}
}

// TestActionAuthorFCAFPipelineRunsTheWalletActions builds a two-scenario
// subset for a wallet that lacks a role of one of them: the other one runs
// with the wallet's own actions on the chosen device, and the first is
// listed as skipped.
func TestActionAuthorFCAFPipelineRunsTheWalletActions(t *testing.T) {
	scope, err := activities.LoadFCAFScope(actionauthor.FCAFRelyingPartySuite)
	require.NoError(t, err)
	scenarioRoles := func(name string) []string {
		scenario, err := walletroles.LoadScenario(
			filepath.Join(scope.Root, scope.Suite, "scenarios", name),
		)
		require.NoError(t, err)
		roles := []string{}
		for _, step := range scenario.Steps {
			with, _ := step["with"].(map[string]any)
			if id, _ := with["action_id"].(string); id != "" {
				roles = append(roles, strings.TrimPrefix(id, scope.Contract.ReferencePrefix))
			}
		}
		return roles
	}
	kept := scenarioRoles(fcafPipelineHaipVP)
	// A scenario with a role the kept scenario does not run.
	paths, err := filepath.Glob(filepath.Join(scope.Root, scope.Suite, "scenarios", "*.yaml"))
	require.NoError(t, err)
	skipped, missing := "", ""
	for _, path := range paths {
		for _, role := range scenarioRoles(filepath.Base(path)) {
			if !slices.Contains(kept, role) {
				skipped, missing = filepath.Base(path), role
				break
			}
		}
		if skipped != "" {
			break
		}
	}
	require.NotEmpty(t, skipped)

	enqueued := stubFCAFPipelineQueue(t)
	var response ActionAuthorFCAFPipelineResponse
	(&tests.ApiScenario{
		Name:   "subset with a missing role",
		Method: http.MethodPost,
		URL:    fcafPipelineURL,
		Body: strings.NewReader(
			fcafPipelineBody(fcafPipelineDevice, fcafPipelineHaipVP, skipped),
		),
		Headers: map[string]string{
			"Authorization": "Bearer " + authoringToken(t, "userA"),
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"status":"queued"`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return setupFCAFPipelineApp(t, missing)
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			decodeBody(t, res, &response)
			record, err := app.FindFirstRecordByData(
				"pipelines",
				"name",
				actionauthor.FCAFPipelineName("Authoring handler wallet", 2),
			)
			require.NoError(t, err)
			require.False(t, record.GetBool("published"))
			require.Equal(t, authoringNamespace+"/"+record.GetString("canonified_name"), response.Pipeline)
			yaml := record.GetString("yaml")
			require.Contains(t, yaml, "global_device_id: "+fcafPipelineDevice)
			require.Contains(t, yaml, "version_id: "+authoringAndroidVersion)
			require.Contains(t, yaml, "action_id: usera-s-organization/authoring-handler-wallet/")
			require.NotContains(t, yaml, scope.Contract.ReferencePrefix)
		},
	}).Test(t)
	require.Equal(t, fcafPipelineDevice, response.DeviceID)
	require.Equal(t, string(workflowengine.MobileDeviceSemaphoreRunQueued), response.Status)
	require.NotEmpty(t, response.TicketID)
	require.Positive(t, response.TestCount)
	require.Len(t, response.Skipped, 1)
	require.Equal(t, skipped, response.Skipped[0].Scenario)
	require.Contains(t, response.Skipped[0].MissingRoles, missing)
	require.Len(t, *enqueued, 1)
	require.Equal(t, response.Pipeline, (*enqueued)[0].PipelineIdentifier)
}
