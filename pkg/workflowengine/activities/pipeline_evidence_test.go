// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/eudi-conformance-evidence/pkg/credoffer"
	"github.com/forkbombeu/eudi-conformance-evidence/pkg/discovery"
	"github.com/forkbombeu/eudi-conformance-evidence/pkg/presentation"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const (
	evidenceTestCredentialID = testOrgANamespace + "/issuer-123/cred123"
	evidenceTestUseCaseID    = testOrgANamespace + "/verifier-123/usecase123"
	evidenceTestUseCaseCode  = "use-case-stepci-code"
)

// evidenceFixture is an issuer and verifier server plus the Credimi records of
// one credential and one use case verification whose deeplinks point at it.
type evidenceFixture struct {
	server *httptest.Server
}

// newEvidenceFixture serves the external issuer and verifier endpoints that
// evidence extraction reads and stores a credential whose offer and a use case
// verification whose StepCI code point at them. Evidence extraction must never
// call Credimi, so any /api/ request fails the test.
func newEvidenceFixture(t *testing.T, app *tests.TestApp) evidenceFixture {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/issuer/.well-known/openid-credential-issuer":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"credential_issuer":"issuer-1"}`)
		case "/request.jwt":
			w.Header().Set("Content-Type", "application/jwt")
			_, _ = fmt.Fprint(w, "eyJhbGciOiJFUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.c2ln")
		default:
			if strings.HasPrefix(r.URL.Path, "/api/") {
				t.Errorf("evidence extraction called Credimi: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	fixture := evidenceFixture{server: server}

	issuer := createTestIssuer(t, app, testOrgAID, "Issuer 123", "https://issuer.example")
	credential := createTestCredential(t, app, testOrgAID, issuer.Id, "cred123")
	credential.Set("deeplink", fixture.credentialOffer())
	require.NoError(t, app.Save(credential))
	createTestUseCaseVerification(t, app, "usecase123", evidenceTestUseCaseCode, "")
	return fixture
}

// credentialOffer is a credential offer deeplink of the fixture issuer.
func (f evidenceFixture) credentialOffer() string {
	offer := url.QueryEscape(fmt.Sprintf(`{"credential_issuer":"%s/issuer"}`, f.server.URL))
	return "openid-credential-offer://?credential_offer=" + offer
}

// verificationDeeplink is the verification deeplink the use case StepCI code
// captures.
func (f evidenceFixture) verificationDeeplink() string {
	return "haip-vp://?request_uri=" + url.QueryEscape(f.server.URL+"/request.jwt") +
		"&request_uri_method=get"
}

// stepCI stands in for StepCI: the use case code captures the verification
// deeplink.
func (f evidenceFixture) stepCI(
	_ context.Context,
	code string,
	_ map[string]string,
) (string, error) {
	if code != evidenceTestUseCaseCode {
		return "", fmt.Errorf("unexpected stepci code %q", code)
	}
	return f.verificationDeeplink(), nil
}

// activity returns an extraction activity whose StepCI runs are served by the
// fixture.
func (f evidenceFixture) activity(app core.App) *PipelineEvidenceExtractionActivity {
	act := NewPipelineEvidenceExtractionActivity(app)
	act.runStepCI = f.stepCI
	return act
}

func evidenceTestInput(workflowID, runID string) PipelineEvidenceExtractionInput {
	return PipelineEvidenceExtractionInput{
		WorkflowDefinition: evidenceTestDefinition(evidenceTestCredentialID, evidenceTestUseCaseID),
		WorkflowID:         workflowID,
		RunID:              runID,
	}
}

func evidenceTestDefinition(credentialID, useCaseID string) *pipelineinternal.WorkflowDefinition {
	return &pipelineinternal.WorkflowDefinition{
		Name: "evidence-pipeline",
		Steps: []pipelineinternal.StepDefinition{
			{
				StepSpec: pipelineinternal.StepSpec{
					ID:  "cred-step",
					Use: "credential-offer",
					With: pipelineinternal.StepInputs{
						Payload: map[string]any{"credential_id": credentialID},
					},
				},
			},
			{
				StepSpec: pipelineinternal.StepSpec{
					ID:  "vp-step",
					Use: "use-case-verification-deeplink",
					With: pipelineinternal.StepInputs{
						Payload: map[string]any{"use_case_id": useCaseID},
					},
				},
			},
		},
	}
}

func TestPipelineEvidenceExtractionActivityExecute(t *testing.T) {
	app := newPipelineResultsTestApp(t)
	fixture := newEvidenceFixture(t, app)
	record := createTestPipelineResult(t, app, "wf-evidence", "run-evidence")

	res, err := fixture.activity(app).Execute(
		t.Context(),
		workflowengine.ActivityInput{Payload: evidenceTestInput("wf-evidence", "run-evidence")},
	)
	require.NoError(t, err)

	out, ok := res.Output.(PipelineEvidenceExtractionOutput)
	require.True(t, ok)
	require.Empty(t, out.Warnings)
	require.Len(t, out.CredentialOffers, 1)
	require.Equal(t, "cred-step", out.CredentialOffers[0]["step_id"])
	require.Equal(t, evidenceTestCredentialID, out.CredentialOffers[0]["credential_id"])
	require.Len(t, out.CredentialWellKnowns, 1)
	require.Equal(t, "cred-step", out.CredentialWellKnowns[0]["step_id"])
	require.Equal(t, evidenceTestCredentialID, out.CredentialWellKnowns[0]["credential_id"])
	require.Equal(
		t,
		map[string]any{"credential_issuer": "issuer-1"},
		out.CredentialWellKnowns[0]["well_known"],
	)
	require.Len(t, out.PresentationResults, 1)
	require.Equal(t, "vp-step", out.PresentationResults[0]["step_id"])
	require.Equal(t, evidenceTestUseCaseID, out.PresentationResults[0]["use_case_id"])
	result, ok := out.PresentationResults[0]["result"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "jwt", result["format"])
	require.Equal(t, fixture.verificationDeeplink(), result["deeplink_uri"])
	require.Equal(t, map[string]any{"sub": "test"}, result["payload"])
	require.Equal(t, "c2ln", result["signature"])
	require.Equal(t, true, result["signature_present"])

	reloaded, err := app.FindRecordById("pipeline_results", record.Id)
	require.NoError(t, err)
	var storedWellKnowns []map[string]any
	require.NoError(
		t,
		json.Unmarshal([]byte(reloaded.GetString("credential_well_knowns")), &storedWellKnowns),
	)
	require.Len(t, storedWellKnowns, 1)
	require.Equal(t, "cred-step", storedWellKnowns[0]["step_id"])
	var storedPresentations []map[string]any
	require.NoError(
		t,
		json.Unmarshal([]byte(reloaded.GetString("presentation_results")), &storedPresentations),
	)
	require.Len(t, storedPresentations, 1)
	require.Equal(t, "vp-step", storedPresentations[0]["step_id"])
}

func TestPipelineEvidenceExtractionActivityRunsDynamicCredentialCode(t *testing.T) {
	app := newPipelineResultsTestApp(t)
	fixture := newEvidenceFixture(t, app)
	issuer := createTestIssuer(t, app, testOrgAID, "Issuer 456", "https://issuer.example")
	credential := createTestCredential(t, app, testOrgAID, issuer.Id, "cred456")
	credential.Set("yaml", "credential-stepci-code")
	credential.Set("secrets", "token: credential-secret\n")
	require.NoError(t, app.Save(credential))

	var gotCode string
	var gotSecrets map[string]string
	act := NewPipelineEvidenceExtractionActivity(app)
	act.runStepCI = func(
		_ context.Context,
		code string,
		secrets map[string]string,
	) (string, error) {
		gotCode, gotSecrets = code, secrets
		return fixture.credentialOffer(), nil
	}
	input := PipelineEvidenceExtractionInput{
		WorkflowDefinition: &pipelineinternal.WorkflowDefinition{
			Name: "dynamic",
			Steps: evidenceTestDefinition(
				testOrgANamespace+"/issuer-456/cred456",
				evidenceTestUseCaseID,
			).Steps[:1],
		},
	}

	res, err := act.Execute(t.Context(), workflowengine.ActivityInput{Payload: input})
	require.NoError(t, err)

	out, ok := res.Output.(PipelineEvidenceExtractionOutput)
	require.True(t, ok)
	require.Equal(t, "credential-stepci-code", gotCode)
	require.Equal(t, map[string]string{"token": "credential-secret"}, gotSecrets)
	require.Len(t, out.CredentialWellKnowns, 1)
	require.Equal(t, "cred-step", out.CredentialWellKnowns[0]["step_id"])
}

// newEvidenceStoreTestActivity returns a fixture activity whose storage
// retries do not wait and run onRetry before each retry.
func newEvidenceStoreTestActivity(
	app core.App,
	fixture evidenceFixture,
	onRetry func(attempt int),
) (*PipelineEvidenceExtractionActivity, *int) {
	retries := 0
	act := fixture.activity(app)
	act.storeRetrySleep = func(ctx context.Context, _ time.Duration) error {
		retries++
		if onRetry != nil {
			onRetry(retries)
		}
		return ctx.Err()
	}
	return act, &retries
}

func executeEvidenceExtraction(
	t *testing.T,
	act *PipelineEvidenceExtractionActivity,
	workflowID, runID string,
) PipelineEvidenceExtractionOutput {
	t.Helper()
	res, err := act.Execute(
		t.Context(),
		workflowengine.ActivityInput{Payload: evidenceTestInput(workflowID, runID)},
	)
	require.NoError(t, err)
	out, ok := res.Output.(PipelineEvidenceExtractionOutput)
	require.True(t, ok)
	require.Len(t, out.CredentialWellKnowns, 1)
	return out
}

func TestPipelineEvidenceExtractionActivityStorageWarnings(t *testing.T) {
	app := newPipelineResultsTestApp(t)
	fixture := newEvidenceFixture(t, app)

	tests := []struct {
		name       string
		workflowID string
		runID      string
		warning    string
		retries    int
	}{
		{
			name:    "missing run ids",
			warning: "pipeline evidence storage skipped: missing workflow_id or run_id",
		},
		{
			name:       "pipeline result never created",
			workflowID: "wf-unknown",
			runID:      "run-unknown",
			warning:    "pipeline evidence storage failed: after 5 attempts: pipeline result not found",
			retries:    4,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			act, retries := newEvidenceStoreTestActivity(app, fixture, nil)
			out := executeEvidenceExtraction(t, act, tc.workflowID, tc.runID)
			require.Len(t, out.Warnings, 1)
			require.Contains(t, out.Warnings[0], tc.warning)
			require.Equal(t, tc.retries, *retries)
		})
	}
}

func TestPipelineEvidenceExtractionActivityStorageRetry(t *testing.T) {
	t.Run("pipeline result created after first attempt", func(t *testing.T) {
		app := newPipelineResultsTestApp(t)
		fixture := newEvidenceFixture(t, app)
		var record *core.Record
		act, retries := newEvidenceStoreTestActivity(app, fixture, func(attempt int) {
			if attempt == 1 {
				record = createTestPipelineResult(t, app, "wf-late", "run-late")
			}
		})

		out := executeEvidenceExtraction(t, act, "wf-late", "run-late")
		require.Empty(t, out.Warnings)
		require.Equal(t, 1, *retries)
		reloaded, err := app.FindRecordById("pipeline_results", record.Id)
		require.NoError(t, err)
		require.Contains(t, reloaded.GetString("credential_well_knowns"), "cred-step")
	})

	t.Run("database save fails transiently", func(t *testing.T) {
		app := newPipelineResultsTestApp(t)
		fixture := newEvidenceFixture(t, app)
		record := createTestPipelineResult(t, app, "wf-flaky", "run-flaky")
		failures := 2
		app.OnRecordUpdate("pipeline_results").BindFunc(func(e *core.RecordEvent) error {
			if failures > 0 {
				failures--
				return errors.New("database is locked")
			}
			return e.Next()
		})
		act, retries := newEvidenceStoreTestActivity(app, fixture, nil)

		out := executeEvidenceExtraction(t, act, "wf-flaky", "run-flaky")
		require.Empty(t, out.Warnings)
		require.Equal(t, 2, *retries)
		reloaded, err := app.FindRecordById("pipeline_results", record.Id)
		require.NoError(t, err)
		require.Contains(t, reloaded.GetString("presentation_results"), "vp-step")
	})

	t.Run("canceled context stops retrying", func(t *testing.T) {
		app := newPipelineResultsTestApp(t)
		ctx, cancel := context.WithCancel(t.Context())
		act := NewPipelineEvidenceExtractionActivity(app)
		out := PipelineEvidenceExtractionOutput{
			CredentialWellKnowns: []map[string]any{{"step_id": "cred-step"}},
		}
		cancel()

		start := time.Now()
		act.storeEvidence(
			ctx,
			PipelineEvidenceExtractionInput{WorkflowID: "wf-gone", RunID: "run-gone"},
			&out,
		)
		require.Less(t, time.Since(start), time.Second)
		require.Len(t, out.Warnings, 1)
		require.Contains(t, out.Warnings[0], "pipeline result not found")
		require.Contains(t, out.Warnings[0], "retry canceled: context canceled")
	})
}

func TestPipelineEvidenceExtractionActivityDeeplinkFailures(t *testing.T) {
	app := newPipelineResultsTestApp(t)
	newEvidenceFixture(t, app)
	act := NewPipelineEvidenceExtractionActivity(app)
	act.runStepCI = func(context.Context, string, map[string]string) (string, error) {
		return "", errors.New("stepci checks failed")
	}

	tests := []struct {
		name         string
		credentialID string
		inputErrors  map[string]string
		want         []string
	}{
		{
			name:         "unresolved inputs and failing stepci",
			credentialID: "${{ earlier.outputs.id }}",
			inputErrors:  map[string]string{"cred-step": "ref not found: earlier.outputs.id"},
			want: []string{
				"failed to resolve credential deeplink for step cred-step: ref not found: earlier.outputs.id",
				"failed to resolve verification deeplink for step vp-step: stepci checks failed",
			},
		},
		{
			name:         "missing credential record",
			credentialID: testOrgANamespace + "/issuer-123/missing",
			want: []string{
				"failed to resolve credential deeplink for step cred-step: credimi record not found",
				"failed to resolve verification deeplink for step vp-step: stepci checks failed",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := PipelineEvidenceExtractionInput{
				WorkflowDefinition: evidenceTestDefinition(tc.credentialID, evidenceTestUseCaseID),
				InputErrors:        tc.inputErrors,
			}
			res, err := act.Execute(t.Context(), workflowengine.ActivityInput{Payload: input})
			require.NoError(t, err)

			out, ok := res.Output.(PipelineEvidenceExtractionOutput)
			require.True(t, ok)
			require.Empty(t, out.CredentialOffers)
			require.Empty(t, out.PresentationResults)
			require.Len(t, out.Warnings, len(tc.want)+1)
			for i, want := range tc.want {
				require.Contains(t, out.Warnings[i], want)
			}
			require.Equal(
				t,
				"no credential well-knowns or presentation results were extracted",
				out.Warnings[len(tc.want)],
			)
		})
	}
}

func TestPipelineEvidenceExtractionActivityWarnsWhenEmpty(t *testing.T) {
	act := NewPipelineEvidenceExtractionActivity(newPipelineResultsTestApp(t))
	res, err := act.Execute(
		t.Context(),
		workflowengine.ActivityInput{
			Payload: PipelineEvidenceExtractionInput{
				WorkflowDefinition: &pipelineinternal.WorkflowDefinition{Name: "empty"},
			},
		},
	)
	require.NoError(t, err)

	out, ok := res.Output.(PipelineEvidenceExtractionOutput)
	require.True(t, ok)
	require.Empty(t, out.CredentialWellKnowns)
	require.Empty(t, out.CredentialOffers)
	require.Empty(t, out.PresentationResults)
	require.Equal(
		t,
		[]string{"no credential well-knowns or presentation results were extracted"},
		out.Warnings,
	)
}

func TestPipelineEvidenceHelpers(t *testing.T) {
	require.Nil(t, decodeRawJSON(nil))
	require.Equal(t, "not-json", decodeRawJSON(json.RawMessage("not-json")))

	credentialErr := &credoffer.ExtractionError{}
	credentialErr.Error.Message = "credential failed"
	presentationErr := &presentation.ExtractionError{}
	presentationErr.Error.Message = "presentation failed"

	warnings := []string{}
	appendCredentialWarning(&warnings, discovery.Step{StepID: "credential-step"}, nil)
	appendCredentialWarning(
		&warnings,
		discovery.Step{StepID: "credential-step"},
		&credoffer.Result{Error: credentialErr},
	)
	appendPresentationWarning(&warnings, discovery.Step{StepID: "presentation-step"}, nil)
	appendPresentationWarning(
		&warnings,
		discovery.Step{StepID: "presentation-step"},
		&presentation.Result{Error: presentationErr},
	)

	require.Contains(t, warnings, "failed to extract credential evidence for step credential-step")
	require.Contains(
		t,
		warnings,
		"failed to extract credential evidence for step credential-step: credential failed",
	)
	require.Contains(
		t,
		warnings,
		"failed to extract presentation evidence for step presentation-step",
	)
	require.Contains(
		t,
		warnings,
		"failed to extract presentation evidence for step presentation-step: presentation failed",
	)
}
