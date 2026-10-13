// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/activity"
	temporalmocks "go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/testsuite"
)

const (
	fcafTestNamespace  = "default-test-namespace"
	fcafTestWorkflowID = "default-test-workflow-id"
	fcafTestRunID      = "default-test-run-id"
	fcafTestTestID     = "WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001"
	fcafTestSource     = "pipeline.pid.presentation.sdjwt.all-claims"
)

type fcafHistoryIterator struct {
	events []*historypb.HistoryEvent
	index  int
}

func (f *fcafHistoryIterator) HasNext() bool { return f.index < len(f.events) }

func (f *fcafHistoryIterator) Next() (*historypb.HistoryEvent, error) {
	if f.index >= len(f.events) {
		return nil, errors.New("no more events")
	}
	event := f.events[f.index]
	f.index++
	return event, nil
}

func setFCAFTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv(
		temporalcrypto.SecretsEncryptionKeyEnv,
		"MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDA=",
	)
	t.Cleanup(temporalclient.ClearTestClients)
}

func fcafTestOutputKind(use string) (workflowengine.OutputKind, bool) {
	if use == "http-request" {
		return workflowengine.OutputMap, true
	}
	return 0, false
}

func fcafPayloads(t *testing.T, value any) *commonpb.Payloads {
	t.Helper()
	payloads, err := temporalcrypto.DataConverter().ToPayloads(value)
	require.NoError(t, err)
	return payloads
}

// fcafHistory records one http-request step "obtain" whose output body carries pidSDJWT.
func fcafHistory(t *testing.T, pidSDJWT string) []*historypb.HistoryEvent {
	t.Helper()
	def := &pipelineinternal.WorkflowDefinition{
		Name: "fcaf",
		Steps: []pipelineinternal.StepDefinition{
			{StepSpec: pipelineinternal.StepSpec{ID: "obtain", Use: "http-request"}},
		},
	}
	return []*historypb.HistoryEvent{
		{
			EventId:   1,
			EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
			Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
				WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
					Input: fcafPayloads(t, map[string]any{"workflow_definition": def}),
				},
			},
		},
		{
			EventId:   5,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
			Attributes: &historypb.HistoryEvent_ActivityTaskScheduledEventAttributes{
				ActivityTaskScheduledEventAttributes: &historypb.ActivityTaskScheduledEventAttributes{
					Input: fcafPayloads(t, workflowengine.ActivityInput{
						Config: map[string]string{workflowengine.StepIDConfigKey: "obtain"},
					}),
				},
			},
		},
		{
			EventId:   7,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
			Attributes: &historypb.HistoryEvent_ActivityTaskCompletedEventAttributes{
				ActivityTaskCompletedEventAttributes: &historypb.ActivityTaskCompletedEventAttributes{
					ScheduledEventId: 5,
					Result: fcafPayloads(t, workflowengine.ActivityResult{
						Output: map[string]any{"body": map[string]any{"pid_sdjwt": pidSDJWT}},
					}),
				},
			},
		},
	}
}

func mockFCAFHistory(t *testing.T, events []*historypb.HistoryEvent) *temporalmocks.Client {
	t.Helper()
	c := &temporalmocks.Client{}
	c.On(
		"GetWorkflowHistory",
		mock.Anything,
		fcafTestWorkflowID,
		fcafTestRunID,
		false,
		enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	).Return(&fcafHistoryIterator{events: events})
	temporalclient.SetClientForTests(fcafTestNamespace, c)
	return c
}

func executeFCAFValidation(
	t *testing.T,
	app core.App,
	input workflowengine.ActivityInput,
) (FCAFValidationActivityOutput, error) {
	t.Helper()
	act := NewFCAFValidationActivity(app, fcafTestOutputKind)
	env := (&testsuite.WorkflowTestSuite{}).NewTestActivityEnvironment()
	env.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{Name: act.Name()})

	encoded, err := env.ExecuteActivity(act.Name(), input)
	if err != nil {
		return FCAFValidationActivityOutput{}, err
	}
	var result struct {
		Output FCAFValidationActivityOutput `json:"output"`
	}
	require.NoError(t, encoded.Get(&result))
	return result.Output, nil
}

func TestFCAFValidationActivityResolvesFromHistoryAndStoresFullReport(t *testing.T) {
	setFCAFTestEnv(t)
	pidSDJWT := `{"query_0":["` + testPIDSDJWTPresentation(t) + `"]}`
	historyClient := mockFCAFHistory(t, fcafHistory(t, pidSDJWT))
	app := newPipelineResultsTestApp(t)
	ensureFCAFReportFields(t, app)
	record := createTestPipelineResult(t, app, fcafTestWorkflowID, fcafTestRunID)

	output, err := executeFCAFValidation(t, app, workflowengine.ActivityInput{
		Payload: FCAFValidationActivityInput{
			TestIDs: []string{fcafTestTestID},
			Pipeline: map[string]any{
				fcafTestSource: map[string]any{
					"output": map[string]any{
						"pid_sdjwt": "${{ obtain.outputs.body.pid_sdjwt }}",
					},
				},
			},
		},
	})
	require.NoError(t, err)
	historyClient.AssertExpectations(t)

	reloaded, err := app.FindRecordById("pipeline_results", record.Id)
	require.NoError(t, err)
	storedJSON := readPipelineResultFile(t, app, reloaded, "fcaf_report")
	var storedReport map[string]any
	require.NoError(t, json.Unmarshal(storedJSON, &storedReport))
	storedEvidence := storedReport["evidence"].(map[string]any)["pid_sdjwt"].(map[string]any)
	require.NotNil(t, storedEvidence["value"])
	storedSum := sha256.Sum256(storedJSON)
	require.Equal(t, hex.EncodeToString(storedSum[:]), output.ReportSHA256)

	require.Equal(t, "passed", output.Report.Status)
	require.Len(t, output.Report.ExecutedTests, 1)
	require.Contains(t, output.Report.Evidence, "pid_sdjwt")
	compact, err := json.Marshal(output.Report)
	require.NoError(t, err)
	require.NotContains(t, string(compact), `"value"`)
	require.NotContains(t, string(compact), `"tests"`)

	leafJSON, err := json.Marshal(pidSDJWT)
	require.NoError(t, err)
	sum := sha256.Sum256(leafJSON)
	require.Equal(t, []FCAFEvidenceReference{{
		Source:  fcafTestSource,
		Path:    "output.pid_sdjwt",
		Ref:     "obtain.outputs.body.pid_sdjwt",
		StepID:  "obtain",
		EventID: 7,
		SHA256:  hex.EncodeToString(sum[:]),
	}}, output.EvidenceIndex)
}

func TestFCAFValidationActivityStoresChildPipelineReportOnRootRun(t *testing.T) {
	setFCAFTestEnv(t)
	historyClient := &temporalmocks.Client{}
	temporalclient.SetClientForTests(fcafTestNamespace, historyClient)
	app := newPipelineResultsTestApp(t)
	ensureFCAFReportFields(t, app)
	root := createTestPipelineResult(t, app, "root-wf", "root-run")

	output, err := executeFCAFValidation(t, app, workflowengine.ActivityInput{
		Payload: FCAFValidationActivityInput{
			TestIDs: []string{fcafTestTestID},
			Pipeline: map[string]any{
				fcafTestSource: map[string]any{
					"output": map[string]any{
						"pid_sdjwt": `{"query_0":["` + testPIDSDJWTPresentation(t) + `"]}`,
					},
				},
			},
		},
		Config: map[string]string{
			workflowengine.TelemetryRootWorkflowIDKey: "root-wf",
			workflowengine.TelemetryRootRunIDKey:      "root-run",
		},
	})
	require.NoError(t, err)
	historyClient.AssertNotCalled(t, "GetWorkflowHistory")
	require.Empty(t, output.EvidenceIndex)
	reloaded, err := app.FindRecordById("pipeline_results", root.Id)
	require.NoError(t, err)
	require.Len(t, reloaded.GetStringSlice("fcaf_report"), 1)
}

func TestFCAFValidationActivityFailsWhenReportStorageFails(t *testing.T) {
	setFCAFTestEnv(t)
	temporalclient.SetClientForTests(fcafTestNamespace, &temporalmocks.Client{})

	_, err := executeFCAFValidation(t, newPipelineResultsTestApp(t), workflowengine.ActivityInput{
		Payload: FCAFValidationActivityInput{
			TestIDs:  []string{fcafTestTestID},
			Pipeline: map[string]any{fcafTestSource: map[string]any{"output": map[string]any{}}},
		},
	})
	require.ErrorContains(t, err, "store FCAF report")
	require.ErrorContains(t, err, "pipeline result not found")
	requireActivityError(t, err, errorcodes.RecordNotFound, true)
}

func TestNormalizeValidationTestIDsSupportsBatchAndLegacyInputs(t *testing.T) {
	ids := normalizeValidationTestIDs(FCAFValidationActivityInput{
		TestID:  " test-legacy ",
		TestIDs: []string{"test-one", "test-one", " ", "test-two"},
	})

	require.Equal(t, []string{"test-one", "test-two", "test-legacy"}, ids)
}

func TestNormalizeValidationTestIDsRejectsEmptyInputs(t *testing.T) {
	require.Empty(t, normalizeValidationTestIDs(FCAFValidationActivityInput{}))
}

func TestFCAFValidationActivityRequiresAggregateOutput(t *testing.T) {
	act := NewFCAFValidationActivity(nil, fcafTestOutputKind)

	_, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: FCAFValidationActivityInput{TestID: "test-one"},
	})

	require.Error(t, err)
}

func testPIDSDJWTPresentation(t *testing.T) string {
	t.Helper()

	header := map[string]any{"alg": "none"}
	payload := map[string]any{
		"vct":               "urn:eudi:pid:1",
		"iss":               "https://issuer.example.test",
		"family_name":       "Trotter",
		"given_name":        "Filippo",
		"birthdate":         "1999-11-01",
		"place_of_birth":    map[string]any{"country": "IT"},
		"nationalities":     []string{"IT"},
		"date_of_expiry":    "2026-10-11",
		"issuing_authority": "GR Administrative authority",
		"issuing_country":   "GR",
		"email":             "person@example.test",
	}

	return encodeJWTLikeSegment(t, header) + "." + encodeJWTLikeSegment(t, payload) + ".~"
}

func encodeJWTLikeSegment(t *testing.T, value map[string]any) string {
	t.Helper()
	data, err := json.Marshal(value)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(data)
}
