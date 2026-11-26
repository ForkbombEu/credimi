// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"path"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/require"
)

// authoringWalletPath is the path of the wallet of authoringAndroidVersion.
const authoringWalletPath = "usera-s-organization/authoring-handler-wallet"

// fixSessionBody starts an FCAF fix session on the admin runner's emulator.
func fixSessionBody(mode string) string {
	return `{"device_id":"` + authoringAdminRunner + `/android-emulator",` +
		`"wallet_version_identifier":"` + authoringAndroidVersion + `",` +
		`"mode":"` + mode + `","fcaf_fix":true}`
}

// setupFixSessionApp is setupActionAuthorApp with a current FCAF action of
// every role on the authoring wallet and, with failing, a finished run of
// the wallet's FCAF pipeline in which the action of the role failing failed.
func setupFixSessionApp(t testing.TB, failing string) *tests.TestApp {
	t.Helper()
	app := setupActionAuthorApp(t)
	wallet, err := app.FindFirstRecordByData("wallets", "name", "Authoring handler wallet")
	require.NoError(t, err)
	actions := saveFCAFActions(t, app, wallet.GetString("owner"), wallet.Id)
	pipelines, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	pipelineRecord := core.NewRecord(pipelines)
	pipelineRecord.Set("owner", wallet.GetString("owner"))
	pipelineRecord.Set("name", actionauthor.FCAFPipelineName(authoringWalletPath, nil))
	pipelineRecord.Set("description", "fcaf")
	pipelineRecord.Set("yaml", "name: x\n")
	require.NoError(t, app.Save(pipelineRecord))
	results, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	if results.Fields.GetByName("fcaf_report") == nil {
		results.Fields.Add(&core.FileField{Name: "fcaf_report", MaxSelect: 1, MaxSize: 1 << 20})
		require.NoError(t, app.Save(results))
	}
	outcomes := actionauthor.FCAFRoleOutcomes{WalletPrefix: authoringWalletPath + "/"}
	for role, action := range actions {
		outcome := actionauthor.FCAFRoleOutcome{
			Action: authoringWalletPath + "/" + action.GetString("canonified_name"),
			Role:   role,
			Status: actionauthor.FCAFRunOK,
			OK:     1,
		}
		if role == failing {
			outcome.Status, outcome.OK, outcome.Failing = actionauthor.FCAFRunFailing, 0, 1
			outcome.FixtureID = "fcaf-haip-vp.yaml#present"
		}
		outcomes.Roles = append(outcomes.Roles, outcome)
	}
	report, err := json.Marshal(map[string]any{"role_outcomes": outcomes})
	require.NoError(t, err)
	result := core.NewRecord(results)
	result.Set("owner", wallet.GetString("owner"))
	result.Set("pipeline", pipelineRecord.Id)
	result.Set("workflow_id", "Pipeline-fix")
	result.Set("run_id", "run-fix")
	file, err := filesystem.NewFileFromBytes(report, "fcaf-assessment.json")
	require.NoError(t, err)
	result.Set("fcaf_report", []*filesystem.File{file})
	require.NoError(t, app.Save(result))
	return app
}

func TestActionAuthorStartRefusesAFixWithNothingToFix(t *testing.T) {
	token := authoringToken(t, "userA")
	enqueued := stubActionAuthorQueue(t, true)
	for _, scenario := range []struct {
		name    string
		body    string
		failing string
		status  int
		reason  string
	}{
		{
			name:   "fix without the fcaf mode",
			body:   fixSessionBody("chat"),
			status: http.StatusBadRequest,
			reason: `"invalid_request"`,
		},
		{
			name:   "no role failed in the last run",
			body:   fixSessionBody("fcaf"),
			status: http.StatusUnprocessableEntity,
			reason: `"fcaf_nothing_to_fix"`,
		},
	} {
		(&tests.ApiScenario{
			Name:            scenario.name,
			Method:          http.MethodPost,
			URL:             "/api/action-author/sessions",
			Body:            strings.NewReader(scenario.body),
			Headers:         map[string]string{"Authorization": "Bearer " + token},
			ExpectedStatus:  scenario.status,
			ExpectedContent: []string{scenario.reason},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return setupFixSessionApp(t, "")
			},
		}).Test(t)
	}
	require.Empty(t, enqueued.TicketID)
}

func TestActionAuthorStartQueuesAFixSession(t *testing.T) {
	token := authoringToken(t, "userA")
	enqueued := stubActionAuthorQueue(t, true)
	var response ActionAuthorStartResponse
	(&tests.ApiScenario{
		Name:            "fix session of a wallet with a failing role",
		Method:          http.MethodPost,
		URL:             "/api/action-author/sessions",
		Body:            strings.NewReader(fixSessionBody("fcaf")),
		Headers:         map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus:  http.StatusAccepted,
		ExpectedContent: []string{`"status":"queued"`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return setupFixSessionApp(t, "fcaf-expect-request-rejected")
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(body, &response))
			runnerNamespace := authoringRunnerNamespace(t, app, path.Base(authoringAdminRunner))
			user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
			require.NoError(t, err)
			want := actionauthor.SessionGrant{
				SessionWorkflowID: actionauthor.SessionWorkflowID(
					strings.TrimPrefix(runnerNamespace, "runner-"),
					response.TicketID,
				),
				Namespace:               runnerNamespace,
				Organization:            "usera-s-organization",
				DeviceID:                authoringAdminRunner + "/android-emulator",
				Platform:                "android",
				WalletVersionIdentifier: authoringAndroidVersion,
				ActorID:                 user.Id,
				Mode:                    actionauthor.SessionModeFCAF,
				FCAFSuite:               actionauthor.FCAFRelyingPartySuite,
				FCAFFix:                 true,
			}
			ticketGrant, _ := enqueued.ActionAuthor["ticket_grant"].(string)
			require.NoError(t, actionauthor.VerifyTicket(app, ticketGrant, want))
			// The ticket of a fix session is not an authoring session's.
			authoring := want
			authoring.FCAFFix = false
			require.ErrorIs(
				t,
				actionauthor.VerifyTicket(app, ticketGrant, authoring),
				actionauthor.ErrForbidden,
			)
		},
	}).Test(t)
	require.Equal(t, true, enqueued.ActionAuthor["fcaf_fix"])
	require.Equal(t, actionauthor.SessionModeFCAF, enqueued.ActionAuthor["mode"])
}
