// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/fcaf/catalog"
	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/forkbombeu/credimi/pkg/fcaf/validators"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestEngineExecutesGraphBackedTest(t *testing.T) {
	engine, err := New(nil)
	require.NoError(t, err)

	cat := &catalog.Catalog{
		Tests: map[string]dsl.TestDefinition{
			"test-1": {
				ID:                  "test-1",
				Title:               "test",
				Suite:               dsl.Suite{SUT: "wallet_solution", Role: "relying_party"},
				NormativeReferences: []dsl.NormativeReference{{Title: "reference"}},
				Preconditions: []dsl.PreconditionRef{
					{Ref: "pipeline.pid_sdjwt"},
				},
				Evidence: map[string]dsl.EvidenceBinding{
					"pid_sdjwt": {From: "pipeline.pid_sdjwt.outputs.pid_sdjwt"},
				},
				Assertions: []dsl.AssertionDefinition{{
					ID:        "email-present",
					Validator: "sdjwt.claim_present",
					Input:     "evidence.pid_sdjwt",
					Params:    map[string]any{"claim": "email"},
				}},
				Verdict: dsl.VerdictPolicy{PassWhen: "all_assertions_pass"},
			},
		},
		Preconditions: map[string]dsl.PreconditionDefinition{
			"pipeline.pid_sdjwt": {
				ID:         "pipeline.pid_sdjwt",
				Kind:       "pipeline",
				PipelineID: "/org-owner/pid-sdjwt",
				Outputs: map[string]dsl.OutputDefinition{
					"pid_sdjwt": {
						Path:    "$.output.http-get-verifier-backend.eudiw.dev-0007.outputs.body.vp_token.query_0[0]",
						Decoder: "sdjwt.presentation",
					},
				},
			},
		},
	}

	report, err := engine.ExecuteCatalog(
		context.Background(),
		cat,
		[]string{"test-1"},
		"",
		map[string]any{"app_url": "https://credimi.test"},
		evidence.Bundle{
			PipelineOutputs: map[string]any{
				"pipeline.pid_sdjwt": samplePipelineOutput(),
			},
		},
	)

	require.NoError(t, err)
	require.Equal(t, validators.StatusPass, report.Tests[0].Status)
	require.Equal(t, 1, report.Summary.Pass)
	report.PopulateDerivedViews()
	require.Equal(t, "passed", report.Status)
	require.Contains(t, report.Evidence, "pid_sdjwt")
	require.Equal(t, "sdjwt.presentation", report.Evidence["pid_sdjwt"].Type)
	value, ok := report.Evidence["pid_sdjwt"].Value.(map[string]any)
	require.True(t, ok)
	require.NotEmpty(t, value["raw"])
	claims, ok := value["claims"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "person@example.test", claims["email"])
	require.Contains(t, report.Evidence, "pipeline.pid_sdjwt.run")
	runEvidence, ok := report.Evidence["pipeline.pid_sdjwt.run"].Value.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "wf-1", runEvidence["workflow_id"])
	require.Equal(t, "run-1", runEvidence["run_id"])
	require.Equal(t, "https://credimi.test/my/tests/runs/wf-1/run-1", runEvidence["pipeline_url"])
	require.Equal(
		t,
		[]string{"pipeline.pid_sdjwt.run", "pid_sdjwt"},
		report.ExecutedTests[0].Preconditions[0].EvidenceKeys,
	)
	require.Equal(t, []string{"pid_sdjwt"}, report.ExecutedTests[0].Assertions[0].EvidenceKeys)
}

func TestEngineBlocksWhenPipelineOutputMissing(t *testing.T) {
	engine, err := New(nil)
	require.NoError(t, err)

	cat := &catalog.Catalog{
		Tests: map[string]dsl.TestDefinition{
			"test-1": {
				ID:                  "test-1",
				Title:               "test",
				Suite:               dsl.Suite{SUT: "wallet_solution", Role: "relying_party"},
				NormativeReferences: []dsl.NormativeReference{{Title: "reference"}},
				Preconditions:       []dsl.PreconditionRef{{Ref: "pipeline.pid_sdjwt"}},
				Assertions: []dsl.AssertionDefinition{{
					ID:        "email-present",
					Validator: "sdjwt.claim_present",
					Input:     "evidence.pid_sdjwt",
					Params:    map[string]any{"claim": "email"},
				}},
				Verdict: dsl.VerdictPolicy{PassWhen: "all_assertions_pass"},
			},
		},
		Preconditions: map[string]dsl.PreconditionDefinition{
			"pipeline.pid_sdjwt": {
				ID:         "pipeline.pid_sdjwt",
				Kind:       "pipeline",
				PipelineID: "/org-owner/pid-sdjwt",
			},
		},
	}

	report, err := engine.ExecuteCatalog(
		context.Background(),
		cat,
		[]string{"test-1"},
		"",
		nil,
		evidence.Bundle{},
	)

	require.NoError(t, err)
	require.Equal(t, validators.StatusBlocked, report.Tests[0].Status)
}

func TestEngineAcceptsPipelineOutputsKeyedByPipelineID(t *testing.T) {
	engine, err := New(nil)
	require.NoError(t, err)

	cat := &catalog.Catalog{
		Tests: map[string]dsl.TestDefinition{
			"test-1": {
				ID:                  "test-1",
				Title:               "test",
				Suite:               dsl.Suite{SUT: "wallet_solution", Role: "relying_party"},
				NormativeReferences: []dsl.NormativeReference{{Title: "reference"}},
				Preconditions: []dsl.PreconditionRef{
					{Ref: "pipeline.pid_sdjwt"},
				},
				Evidence: map[string]dsl.EvidenceBinding{
					"pid_sdjwt": {From: "pipeline.pid_sdjwt.outputs.pid_sdjwt"},
				},
				Assertions: []dsl.AssertionDefinition{{
					ID:        "email-present",
					Validator: "sdjwt.claim_present",
					Input:     "evidence.pid_sdjwt",
					Params:    map[string]any{"claim": "email"},
				}},
				Verdict: dsl.VerdictPolicy{PassWhen: "all_assertions_pass"},
			},
		},
		Preconditions: map[string]dsl.PreconditionDefinition{
			"pipeline.pid_sdjwt": {
				ID:         "pipeline.pid_sdjwt",
				Kind:       "pipeline",
				PipelineID: "/org-owner/pid-sdjwt",
				Outputs: map[string]dsl.OutputDefinition{
					"pid_sdjwt": {
						Path:    "$.output.http-get-verifier-backend.eudiw.dev-0007.outputs.body.vp_token.query_0[0]",
						Decoder: "sdjwt.presentation",
					},
				},
			},
		},
	}

	report, err := engine.ExecuteCatalog(
		context.Background(),
		cat,
		[]string{"test-1"},
		"",
		nil,
		evidence.Bundle{
			PipelineOutputs: map[string]any{
				"/org-owner/pid-sdjwt": samplePipelineOutput(),
			},
		},
	)

	require.NoError(t, err)
	require.Equal(t, validators.StatusPass, report.Tests[0].Status)
}

func TestEngineAcceptsTypedPipelineExecutionResult(t *testing.T) {
	engine, err := New(nil)
	require.NoError(t, err)

	cat := &catalog.Catalog{
		Tests: map[string]dsl.TestDefinition{
			"test-1": {
				ID:                  "test-1",
				Title:               "test",
				Suite:               dsl.Suite{SUT: "wallet_solution", Role: "relying_party"},
				NormativeReferences: []dsl.NormativeReference{{Title: "reference"}},
				Preconditions:       []dsl.PreconditionRef{{Ref: "pipeline.pid_sdjwt"}},
				Evidence: map[string]dsl.EvidenceBinding{
					"pid_sdjwt": {From: "pipeline.pid_sdjwt.outputs.pid_sdjwt"},
				},
				Assertions: []dsl.AssertionDefinition{{
					ID:        "email-present",
					Validator: "sdjwt.claim_present",
					Input:     "evidence.pid_sdjwt",
					Params:    map[string]any{"claim": "email"},
				}},
				Verdict: dsl.VerdictPolicy{PassWhen: "all_assertions_pass"},
			},
		},
		Preconditions: map[string]dsl.PreconditionDefinition{
			"pipeline.pid_sdjwt": {
				ID:         "pipeline.pid_sdjwt",
				Kind:       "pipeline",
				PipelineID: "/org-owner/pid-sdjwt",
				Outputs: map[string]dsl.OutputDefinition{
					"pid_sdjwt": {
						Path:    "$.output.http-get-verifier-backend.eudiw.dev-0007.outputs.body.vp_token.query_0[0]",
						Decoder: "sdjwt.presentation",
					},
				},
			},
		},
	}

	report, err := engine.ExecuteCatalog(
		context.Background(),
		cat,
		[]string{"test-1"},
		"",
		nil,
		evidence.Bundle{
			PipelineOutputs: map[string]any{
				"pipeline.pid_sdjwt": evidence.PipelineExecutionResult{
					Output:        samplePipelineOutput()["output"],
					WorkflowID:    "wf-1",
					WorkflowRunID: "run-1",
				},
			},
		},
	)

	require.NoError(t, err)
	require.Equal(t, validators.StatusPass, report.Tests[0].Status)
	require.Equal(t, "wf-1", report.Tests[0].Preconditions[0].WorkflowID)
	require.Equal(t, "run-1", report.Tests[0].Preconditions[0].RunID)
}

func TestEnginePipelinePreconditionChecksOnlyRequiredSteps(t *testing.T) {
	failure := evidence.PipelineStepFailure{
		StepID:  "unrelated-test-step",
		Code:    "CRE229",
		Summary: "unrelated assertion failed",
	}
	report := executeRequiredStepsTest(
		t,
		[]string{"required-test-step"},
		[]evidence.PipelineStepFailure{failure},
	)

	require.Equal(t, validators.StatusPass, report.Tests[0].Status)
	require.Contains(t, report.Tests[0].Preconditions[0].Message, "required-test-step")
}

func TestEnginePipelinePreconditionFailsRequiredStep(t *testing.T) {
	failure := evidence.PipelineStepFailure{
		StepID:  "required-test-step",
		Code:    "CRE229",
		Summary: "screen assertion failed",
	}
	report := executeRequiredStepsTest(
		t,
		[]string{"required-test-step"},
		[]evidence.PipelineStepFailure{failure},
	)

	require.Equal(t, validators.StatusFail, report.Tests[0].Status)
	require.Contains(t, report.Tests[0].Preconditions[0].Message, `"required-test-step" failed`)
	require.Contains(t, report.Tests[0].Preconditions[0].Message, "screen assertion failed")
}

func TestEnginePipelinePreconditionFailsMissingRequiredStep(t *testing.T) {
	report := executeRequiredStepsTest(t, []string{"missing-test-step"}, nil)

	require.Equal(t, validators.StatusFail, report.Tests[0].Status)
	require.Contains(t, report.Tests[0].Preconditions[0].Message, "was not executed")
}

func executeRequiredStepsTest(
	t *testing.T,
	requiredSteps []string,
	failures []evidence.PipelineStepFailure,
) Report {
	t.Helper()
	engine, err := New(nil)
	require.NoError(t, err)
	cat := &catalog.Catalog{
		Tests: map[string]dsl.TestDefinition{
			"test-1": {
				ID:                  "test-1",
				Suite:               dsl.Suite{SUT: "wallet_solution", Role: "relying_party"},
				NormativeReferences: []dsl.NormativeReference{{Title: "reference"}},
				Preconditions:       []dsl.PreconditionRef{{Ref: "pipeline.shared"}},
				Evidence: map[string]dsl.EvidenceBinding{
					"value": {From: "pipeline.shared.outputs.value"},
				},
				Assertions: []dsl.AssertionDefinition{{
					ID:        "value-present",
					Validator: "evidence.present",
					Input:     "evidence.value",
				}},
				Verdict: dsl.VerdictPolicy{PassWhen: "all_assertions_pass"},
			},
		},
		Preconditions: map[string]dsl.PreconditionDefinition{
			"pipeline.shared": {
				ID:            "pipeline.shared",
				Kind:          "pipeline",
				PipelineID:    "owner/shared",
				RequiredSteps: requiredSteps,
				Outputs: map[string]dsl.OutputDefinition{
					"value": {Path: "$.output.required-test-step.outputs"},
				},
			},
		},
	}
	report, err := engine.ExecuteCatalog(
		context.Background(),
		cat,
		[]string{"test-1"},
		"",
		nil,
		evidence.Bundle{PipelineOutputs: map[string]any{
			"owner/shared": evidence.PipelineExecutionResult{
				Output: map[string]any{
					"required-test-step": map[string]any{"outputs": "ok"},
				},
				StepFailures: failures,
			},
		}},
	)
	require.NoError(t, err)
	return report
}

func samplePipelineOutput() map[string]any {
	return map[string]any{
		"output": map[string]any{
			"http-get-verifier-backend.eudiw.dev-0007": map[string]any{
				"outputs": map[string]any{
					"body": map[string]any{
						"vp_token": map[string]any{
							"query_0": []any{
								"eyJhbGciOiJub25lIn0.eyJfc2QiOlsiTmRUemVld0RjZVRJOXNQVGdRdjBRUG1oU1JZaVQ5cnJwOTB3OE5TY2ZCYyJdLCJ2Y3QiOiJ1cm46ZXVkaTpwaWQ6MSIsImlzcyI6Imh0dHBzOi8vaXNzdWVyLmV4YW1wbGUifQ~WyJzYWx0IiwiZW1haWwiLCJwZXJzb25AZXhhbXBsZS50ZXN0Il0~",
							},
						},
					},
				},
			},
		},
		"workflowId":    "wf-1",
		"workflowRunId": "run-1",
	}
}

type countingValidator struct {
	calls *int
}

func (countingValidator) ID() string { return "test.counting" }

func (v countingValidator) Validate(_ context.Context, _ validators.Input) validators.Result {
	*v.calls++
	return validators.Result{Status: validators.StatusPass, Message: "counted"}
}

func TestEngineReusesPreconditionResultAcrossExecutions(t *testing.T) {
	calls := 0
	registry, err := validators.NewRegistry(countingValidator{calls: &calls})
	require.NoError(t, err)
	engine, err := NewWithCaches(registry, evidence.Extract, NewNodeResultCache(8))
	require.NoError(t, err)
	cat := &catalog.Catalog{
		Tests: map[string]dsl.TestDefinition{
			"test-1": {
				ID:                  "test-1",
				Suite:               dsl.Suite{SUT: "wallet_solution", Role: "relying_party"},
				NormativeReferences: []dsl.NormativeReference{{Title: "reference"}},
				Preconditions: []dsl.PreconditionRef{{
					Ref: "assertion.shared",
				}},
				Verdict: dsl.VerdictPolicy{PassWhen: "all_assertions_pass"},
			},
		},
		Preconditions: map[string]dsl.PreconditionDefinition{
			"pipeline.shared": {
				ID:   "pipeline.shared",
				Kind: "pipeline",
				Outputs: map[string]dsl.OutputDefinition{
					"value": {Path: "$.output.step.outputs", Decoder: "raw"},
				},
			},
			"assertion.shared": {
				ID:        "assertion.shared",
				Kind:      "assertion",
				DependsOn: []string{"pipeline.shared"},
				Input:     &dsl.InputBinding{From: "pipeline.shared.outputs.value"},
				Validator: "test.counting",
			},
		},
	}
	bundle := evidence.Bundle{PipelineOutputs: map[string]any{
		"pipeline.shared": evidence.PipelineExecutionResult{
			WorkflowID:    "pipeline-workflow",
			WorkflowRunID: "pipeline-run",
			Output: map[string]any{
				"step": map[string]any{"outputs": "value"},
			},
		},
	}}

	for range 2 {
		report, executeErr := engine.ExecuteCatalog(
			context.Background(),
			cat,
			[]string{"test-1"},
			"",
			nil,
			bundle,
		)
		require.NoError(t, executeErr)
		require.Equal(t, validators.StatusPass, report.Tests[0].Preconditions[0].Status)
	}
	require.Equal(t, 1, calls)
}

func TestDeviceBinding012KBJWTPresentWithLiveEvidenceShape(t *testing.T) {
	engine, err := New(nil)
	require.NoError(t, err)

	cat, err := catalog.Load(
		"../../../config_templates/fcaf/wallet_solution/relying_party",
	)
	require.NoError(t, err)

	tests, err := cat.ResolveSelectedTests(
		[]string{"WS_RP_SM_DeviceBinding__012"},
		"wallet_solution/relying_party",
		nil,
	)
	require.NoError(t, err)
	require.Len(t, tests, 1)

	key := generateP256Key(t)
	jwk := p256PublicJWK(key)
	issuerPayload := map[string]any{
		"_sd_alg": "sha-256",
		"cnf":     map[string]any{"jwk": jwk},
		"vct":     "urn:eudi:pid:1",
	}
	sdjwt := unsignedJWT(t, map[string]any{"alg": "ES256", "typ": "dc+sd-jwt"}, issuerPayload) + "~"
	kbJWT := signKBJWT(t, key, sdjwt)

	fullToken := sdjwt + kbJWT
	report, err := engine.ExecuteCatalog(
		context.Background(),
		cat,
		[]string{"WS_RP_SM_DeviceBinding__012"},
		"",
		nil,
		evidence.Bundle{
			PipelineOutputs: map[string]any{
				"pipeline.pid.presentation.sdjwt.all-claims": evidence.PipelineExecutionResult{
					Output: map[string]any{
						"capture-issuer-pid-dc-sd-jwt-0002":       map[string]any{"outputs": "ok"},
						"getcredential-pid-sdjwt-all-claims-0003": map[string]any{"outputs": "ok"},
						"fake-verifier-pid-sd-jwt-credentials-all-claims-0006": map[string]any{
							"outputs": map[string]any{
								"screenshot_urls": []string{"https://example.test/screenshot1.png"},
							},
						},
						"verifycredential-pid-formeu-issuer-eudiw-dev-0007": map[string]any{"outputs": "ok"},
						"http-get-verifier-backend.eudiw.dev-0008": map[string]any{
							"outputs": map[string]any{
								"body": map[string]any{
									"raw": map[string]any{
										"presentation_response_decrypted": map[string]any{
											"vp_token": map[string]any{
												"query_0": []any{fullToken},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	)

	require.NoError(t, err)
	require.Len(t, report.Tests, 1)
	require.Equal(t, validators.StatusPass, report.Tests[0].Status,
		"case 012 should pass with structurally valid KB-JWT: %s",
		report.Tests[0].Message,
	)
	report.PopulateDerivedViews()
	require.Equal(t, "passed", report.Status)

	require.Len(t, report.Tests[0].Assertions, 3)
	for _, a := range report.Tests[0].Assertions {
		require.Equal(t, validators.StatusPass, a.Status,
			"assertion %s: %s", a.ID, a.Message,
		)
	}
}

func generateP256Key(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return key
}

func p256PublicJWK(key *ecdsa.PrivateKey) map[string]any {
	return map[string]any{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(key.X.FillBytes(make([]byte, 32))),
		"y":   base64.RawURLEncoding.EncodeToString(key.Y.FillBytes(make([]byte, 32))),
	}
}

func signKBJWT(t *testing.T, key *ecdsa.PrivateKey, sdjwt string) string {
	t.Helper()
	digest := sha256.Sum256([]byte(sdjwt))
	claims := jwt.MapClaims{
		"iat":     time.Now().Unix(),
		"aud":     "x509_hash:verifier.example",
		"nonce":   "fcaf-device-binding-012",
		"sd_hash": base64.RawURLEncoding.EncodeToString(digest[:]),
	}
	kbJWT := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	kbJWT.Header["typ"] = "kb+jwt"
	signed, err := kbJWT.SignedString(key)
	require.NoError(t, err)
	return signed
}

func unsignedJWT(t *testing.T, header, payload map[string]any) string {
	t.Helper()
	encode := func(value map[string]any) string {
		raw, err := json.Marshal(value)
		require.NoError(t, err)
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	return encode(header) + "." + encode(payload) + ".c2lnbmF0dXJl"
}
