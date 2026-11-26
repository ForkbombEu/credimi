// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/walletroles"
	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
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
// userB's organization, the scenarios' reference credentials published by
// their own organization and, saved as current FCAF actions of userA's
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
	seedFCAFReferenceCredentials(t, app)
	if without == nil {
		return app
	}
	wallet, err := app.FindFirstRecordByData("wallets", "name", "Authoring handler wallet")
	require.NoError(t, err)
	saveFCAFActions(t, app, wallet.GetString("owner"), wallet.Id, without...)
	return app
}

// seedFCAFReferenceCredentials creates, published, every credential the
// suite's scenarios name (organization/issuer/credential), each under an
// organization and issuer of those names.
func seedFCAFReferenceCredentials(t testing.TB, app *tests.TestApp) []*core.Record {
	t.Helper()
	scope, err := activities.LoadFCAFScope(actionauthor.FCAFRelyingPartySuite)
	require.NoError(t, err)
	paths, err := filepath.Glob(filepath.Join(scope.Root, scope.Suite, "scenarios", "*.yaml"))
	require.NoError(t, err)
	reference := regexp.MustCompile(`credential_id:\s*/?([^/\s]+)/([^/\s]+)/([^/\s]+)\s*$`)
	seen := map[string]bool{}
	var credentials []*core.Record
	find := func(collection, filter string, params dbx.Params) *core.Record {
		record, err := app.FindFirstRecordByFilter(collection, filter, params)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		require.NoError(t, err)
		return record
	}
	save := func(collection string, fields map[string]any) *core.Record {
		c, err := app.FindCollectionByNameOrId(collection)
		require.NoError(t, err)
		record := core.NewRecord(c)
		for key, value := range fields {
			record.Set(key, value)
		}
		require.NoError(t, app.Save(record))
		return record
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		for _, line := range strings.Split(string(data), "\n") {
			match := reference.FindStringSubmatch(line)
			if match == nil || seen[strings.Join(match[1:], "/")] {
				continue
			}
			seen[strings.Join(match[1:], "/")] = true
			org := find("organizations", "canonified_name = {:name}", dbx.Params{"name": match[1]})
			if org == nil {
				org = save("organizations", map[string]any{"name": match[1]})
				require.Equal(t, match[1], org.GetString("canonified_name"))
			}
			issuer := find(
				"credential_issuers",
				"owner = {:owner} && canonified_name = {:name}",
				dbx.Params{"owner": org.Id, "name": match[2]},
			)
			if issuer == nil {
				issuer = save("credential_issuers", map[string]any{
					"owner":     org.Id,
					"name":      match[2],
					"url":       "https://" + match[2] + ".example",
					"published": true,
				})
			}
			credential := save("credentials", map[string]any{
				"owner":             org.Id,
				"credential_issuer": issuer.Id,
				"name":              match[3],
				"published":         true,
			})
			require.Equal(t, match[3], credential.GetString("canonified_name"))
			credentials = append(credentials, credential)
		}
	}
	require.NotEmpty(t, credentials)
	return credentials
}

// saveFCAFActions saves, for organization owner on the wallet, a current FCAF
// action of every role of the suite but those in without, keyed by role.
func saveFCAFActions(
	t testing.TB,
	app *tests.TestApp,
	owner, walletID string,
	without ...string,
) map[string]*core.Record {
	t.Helper()
	scope, err := activities.LoadFCAFScope(actionauthor.FCAFRelyingPartySuite)
	require.NoError(t, err)
	actions, err := app.FindCollectionByNameOrId("wallet_actions")
	require.NoError(t, err)
	if actions.Fields.GetByName("authoring_provenance") == nil {
		actions.Fields.Add(&core.JSONField{Name: "authoring_provenance", MaxSize: 65536})
		require.NoError(t, app.Save(actions))
	}
	saved := map[string]*core.Record{}
	for _, role := range scope.Contract.Roles {
		if slices.Contains(without, role.Name) {
			continue
		}
		action := core.NewRecord(actions)
		action.Set("owner", owner)
		action.Set("wallet", walletID)
		action.Set("name", role.Name)
		action.Set("code", "appId: eu.wallet\n---\n- launchApp\n")
		action.Set("category", role.Category)
		action.Set("authoring_provenance", actionauthor.Provenance{
			FCAFRole:       role.Name,
			FCAFRoleSHA256: role.SHA256(),
		})
		require.NoError(t, app.Save(action))
		saved[role.Name] = action
	}
	return saved
}

func fcafPipelineBody(device string, scenarios ...string) string {
	return fcafPipelineVersionBody(authoringAndroidVersion, device, scenarios...)
}

func fcafPipelineVersionBody(version, device string, scenarios ...string) string {
	quoted := make([]string, len(scenarios))
	for i, scenario := range scenarios {
		quoted[i] = `"` + scenario + `"`
	}
	return `{"wallet_version_identifier":"` + version +
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
			name:   "unknown scenario without extension",
			user:   "userA",
			body:   fcafPipelineBody(fcafPipelineDevice, "nope"),
			status: http.StatusUnprocessableEntity,
			reason: "fcaf_unknown_scenario",
		},
		{
			name: "scenario named twice, with and without extension",
			user: "userA",
			body: fcafPipelineBody(
				fcafPipelineDevice,
				fcafPipelineHaipVP,
				strings.TrimSuffix(fcafPipelineHaipVP, ".yaml"),
			),
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
// subset, one named without its .yaml extension, for a wallet that lacks a
// role of one of them: the other one runs with the wallet's own actions on
// the chosen device, and the first is listed as skipped.
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
		Body: strings.NewReader(fcafPipelineBody(
			fcafPipelineDevice,
			strings.TrimSuffix(fcafPipelineHaipVP, ".yaml"),
			skipped,
		)),
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
				actionauthor.FCAFPipelineName(
					"usera-s-organization/authoring-handler-wallet",
					[]string{skipped, fcafPipelineHaipVP},
				),
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

// TestActionAuthorFCAFPipelineRunsTheCallersActionsOnAForeignWallet runs a
// published wallet of another organization with the caller's own actions on
// it, saved under the wallet owner's path with a suffix since the owner has
// actions of the same names; the caller's same-named wallet never lends its
// actions.
func TestActionAuthorFCAFPipelineRunsTheCallersActionsOnAForeignWallet(t *testing.T) {
	stubFCAFPipelineQueue(t)
	var callers map[string]*core.Record
	(&tests.ApiScenario{
		Name:   "foreign published wallet",
		Method: http.MethodPost,
		URL:    fcafPipelineURL,
		Body: strings.NewReader(
			fcafPipelineVersionBody(authoringForeignVersion, fcafPipelineDevice, fcafPipelineHaipVP),
		),
		Headers: map[string]string{
			"Authorization": "Bearer " + authoringToken(t, "userA"),
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"status":"queued"`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupFCAFPipelineApp(t)
			shareAdminRunner(t, app)
			foreign, err := app.FindFirstRecordByData("wallets", "name", "Foreign wallet")
			require.NoError(t, err)
			foreign.Set("published", true)
			require.NoError(t, app.Save(foreign))
			version, err := app.FindFirstRecordByData("wallet_versions", "wallet", foreign.Id)
			require.NoError(t, err)
			version.Set("downloadable", true)
			require.NoError(t, app.Save(version))
			callerOrg, err := app.FindFirstRecordByData(
				"organizations", "name", "userA's organization",
			)
			require.NoError(t, err)
			// The owner's actions come first, so the caller's get a suffix.
			saveFCAFActions(t, app, foreign.GetString("owner"), foreign.Id)
			callers = saveFCAFActions(t, app, callerOrg.Id, foreign.Id)
			// The caller's own wallet of the same name.
			wallets, err := app.FindCollectionByNameOrId("wallets")
			require.NoError(t, err)
			own := core.NewRecord(wallets)
			own.Set("owner", callerOrg.Id)
			own.Set("name", "Foreign wallet")
			require.NoError(t, app.Save(own))
			saveFCAFActions(t, app, callerOrg.Id, own.Id)
			return app
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			record, err := app.FindFirstRecordByData(
				"pipelines",
				"name",
				actionauthor.FCAFPipelineName(
					"userb-s-organization/foreign-wallet",
					[]string{fcafPipelineHaipVP},
				),
			)
			require.NoError(t, err)
			yaml := record.GetString("yaml")
			reset := callers["fcaf-reset-and-onboard"].GetString("canonified_name")
			require.NotEqual(t, "fcaf-reset-and-onboard", reset)
			require.Contains(t, yaml, "action_id: userb-s-organization/foreign-wallet/"+reset+"\n")
			require.NotContains(t, yaml, "usera-s-organization/foreign-wallet")
			require.NotRegexp(t, `foreign-wallet/fcaf-reset-and-onboard\n`, yaml)
		},
	}).Test(t)
}

// requireNoFCAFPipeline fails when any FCAF pipeline record was written.
func requireNoFCAFPipeline(t testing.TB, app *tests.TestApp) {
	t.Helper()
	records, err := app.FindAllRecords(
		"pipelines",
		dbx.Like("name", actionauthor.FCAFPipelineName("", nil)).Match(false, true),
	)
	require.NoError(t, err)
	require.Empty(t, records)
}

// TestActionAuthorFCAFPipelineIsForOwners pins that a member who is not an
// owner of the organization can neither build nor run its FCAF pipelines.
func TestActionAuthorFCAFPipelineIsForOwners(t *testing.T) {
	enqueued := stubFCAFPipelineQueue(t)
	(&tests.ApiScenario{
		Name:   "member who is not an owner",
		Method: http.MethodPost,
		URL:    fcafPipelineURL,
		Body:   strings.NewReader(fcafPipelineBody(fcafPipelineDevice, fcafPipelineHaipVP)),
		Headers: map[string]string{
			"Authorization": "Bearer " + authoringToken(t, "userA"),
			"Content-Type":  "application/json",
		},
		ExpectedStatus: http.StatusForbidden,
		ExpectedContent: []string{
			`"reason":"forbidden"`,
			"only an owner of the organization can build and run its FCAF pipelines",
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupFCAFPipelineApp(t, "no role left out")
			demoteToMember(t, app, "userA@example.org")
			return app
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			requireNoFCAFPipeline(t, app)
		},
	}).Test(t)
	require.Empty(t, *enqueued)
}

// TestActionAuthorFCAFPipelineNeedsUsableCredentials pins that a pipeline
// naming a reference credential the organization can neither own nor use
// published answers 422 fcaf_credential_unavailable with its path, writing
// nothing.
func TestActionAuthorFCAFPipelineNeedsUsableCredentials(t *testing.T) {
	enqueued := stubFCAFPipelineQueue(t)
	issuerPath := ""
	(&tests.ApiScenario{
		Name:   "unpublished reference credential",
		Method: http.MethodPost,
		URL:    fcafPipelineURL,
		Body:   strings.NewReader(fcafPipelineBody(fcafPipelineDevice, fcafPipelineHaipVP)),
		Headers: map[string]string{
			"Authorization": "Bearer " + authoringToken(t, "userA"),
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  http.StatusUnprocessableEntity,
		ExpectedContent: []string{`"reason":"fcaf_credential_unavailable"`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupFCAFPipelineApp(t, "no role left out")
			credentials, err := app.FindAllRecords("credentials")
			require.NoError(t, err)
			for _, credential := range credentials {
				credential.Set("published", false)
				require.NoError(t, app.Save(credential))
			}
			issuer, err := app.FindRecordById(
				"credential_issuers",
				credentials[0].GetString("credential_issuer"),
			)
			require.NoError(t, err)
			org, err := app.FindRecordById("organizations", credentials[0].GetString("owner"))
			require.NoError(t, err)
			issuerPath = org.GetString("canonified_name") + "/" + issuer.GetString("canonified_name")
			return app
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			var body struct {
				Message string `json:"message"`
			}
			decodeBody(t, res, &body)
			require.Contains(t, body.Message, issuerPath+"/")
			requireNoFCAFPipeline(t, app)
		},
	}).Test(t)
	require.Empty(t, *enqueued)
}

// TestActionAuthorFCAFPipelineUsesTheCallersCopiesOfTheReferenceCredentials
// pins that an organization holding its own unpublished copy of a reference
// credential (same issuer and credential names, as `fcaf sync` imports them)
// runs that copy, even when the reference one is not published, while a
// credential it has no copy of stays the published reference one.
func TestActionAuthorFCAFPipelineUsesTheCallersCopiesOfTheReferenceCredentials(t *testing.T) {
	enqueued := stubFCAFPipelineQueue(t)
	var copied, kept []string
	(&tests.ApiScenario{
		Name:   "caller's copies of the reference credentials",
		Method: http.MethodPost,
		URL:    fcafPipelineURL,
		Body:   strings.NewReader(fcafPipelineBody(fcafPipelineDevice)),
		Headers: map[string]string{
			"Authorization": "Bearer " + authoringToken(t, "userA"),
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"status":"queued"`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupActionAuthorApp(t)
			wallet, err := app.FindFirstRecordByData("wallets", "name", "Authoring handler wallet")
			require.NoError(t, err)
			saveFCAFActions(t, app, wallet.GetString("owner"), wallet.Id, "no role left out")
			credentials := seedFCAFReferenceCredentials(t, app)
			require.Greater(t, len(credentials), 1)
			caller, err := app.FindRecordById("organizations", wallet.GetString("owner"))
			require.NoError(t, err)
			for i, credential := range credentials {
				issuer, err := app.FindRecordById(
					"credential_issuers",
					credential.GetString("credential_issuer"),
				)
				require.NoError(t, err)
				org, err := app.FindRecordById("organizations", credential.GetString("owner"))
				require.NoError(t, err)
				reference := org.GetString("canonified_name") + "/" +
					issuer.GetString("canonified_name") + "/" + credential.GetString("canonified_name")
				if i == 0 {
					// No copy: the published reference credential runs.
					kept = append(kept, reference)
					continue
				}
				credential.Set("published", false)
				require.NoError(t, app.Save(credential))
				ownIssuer, err := app.FindFirstRecordByFilter(
					"credential_issuers",
					"owner = {:owner} && canonified_name = {:name}",
					dbx.Params{"owner": caller.Id, "name": issuer.GetString("canonified_name")},
				)
				if errors.Is(err, sql.ErrNoRows) {
					c, cErr := app.FindCollectionByNameOrId("credential_issuers")
					require.NoError(t, cErr)
					ownIssuer = core.NewRecord(c)
					ownIssuer.Set("owner", caller.Id)
					ownIssuer.Set("name", issuer.GetString("name"))
					ownIssuer.Set("url", issuer.GetString("url"))
					require.NoError(t, app.Save(ownIssuer))
				} else {
					require.NoError(t, err)
				}
				c, err := app.FindCollectionByNameOrId("credentials")
				require.NoError(t, err)
				own := core.NewRecord(c)
				own.Set("owner", caller.Id)
				own.Set("credential_issuer", ownIssuer.Id)
				own.Set("name", credential.GetString("name"))
				require.NoError(t, app.Save(own))
				copied = append(copied, reference)
			}
			return app
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			record, err := app.FindFirstRecordByData(
				"pipelines",
				"name",
				actionauthor.FCAFPipelineName("usera-s-organization/authoring-handler-wallet", nil),
			)
			require.NoError(t, err)
			yaml := record.GetString("yaml")
			for _, reference := range copied {
				_, rest, _ := strings.Cut(reference, "/")
				require.NotContains(t, yaml, reference+"\n")
				require.Contains(t, yaml, authoringNamespace+"/"+rest)
			}
			for _, reference := range kept {
				require.Contains(t, yaml, reference)
			}
		},
	}).Test(t)
	require.Len(t, *enqueued, 1)
}
