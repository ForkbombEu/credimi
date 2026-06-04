// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/eudi-conformance-evidence/pkg/credoffer"
	"github.com/forkbombeu/eudi-conformance-evidence/pkg/discovery"
	"github.com/forkbombeu/eudi-conformance-evidence/pkg/presentation"
	"github.com/stretchr/testify/require"
)

func TestPipelineEvidenceExtractionActivityExecute(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/credential/deeplink":
			offer := url.QueryEscape(fmt.Sprintf(`{"credential_issuer":"%s/issuer"}`, server.URL))
			_, _ = fmt.Fprintf(w, "openid-credential-offer://?credential_offer=%s", offer)
		case "/issuer/.well-known/openid-credential-issuer":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"credential_issuer":"issuer-1"}`)
		case "/api/verification/deeplink":
			requestURI := url.QueryEscape(server.URL + "/request.jwt")
			_, _ = fmt.Fprintf(w, "haip-vp://?request_uri=%s&request_uri_method=get", requestURI)
		case "/request.jwt":
			w.Header().Set("Content-Type", "application/jwt")
			_, _ = fmt.Fprint(w, "eyJhbGciOiJFUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.c2ln")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	act := NewPipelineEvidenceExtractionActivity()
	res, err := act.Execute(
		t.Context(),
		workflowengine.ActivityInput{
			Payload: PipelineEvidenceExtractionInput{
				WorkflowDefinition: &pipelineinternal.WorkflowDefinition{
					Name: "evidence-pipeline",
					Steps: []pipelineinternal.StepDefinition{
						{
							StepSpec: pipelineinternal.StepSpec{
								ID:  "cred-step",
								Use: "credential-offer",
								With: pipelineinternal.StepInputs{
									Payload: map[string]any{"credential_id": "tenant/credential-1"},
								},
							},
						},
						{
							StepSpec: pipelineinternal.StepSpec{
								ID:  "vp-step",
								Use: "use-case-verification-deeplink",
								With: pipelineinternal.StepInputs{
									Payload: map[string]any{"use_case_id": "tenant/use-case-1"},
								},
							},
						},
					},
				},
				CredimiBaseURL: server.URL,
			},
		},
	)
	require.NoError(t, err)

	out, ok := res.Output.(PipelineEvidenceExtractionOutput)
	require.True(t, ok)
	require.Empty(t, out.Warnings)
	require.Len(t, out.CredentialOffers, 1)
	require.Equal(t, "cred-step", out.CredentialOffers[0]["step_id"])
	require.Equal(t, "tenant/credential-1", out.CredentialOffers[0]["credential_id"])
	require.Len(t, out.CredentialWellKnowns, 1)
	require.Equal(t, "cred-step", out.CredentialWellKnowns[0]["step_id"])
	require.Equal(t, "tenant/credential-1", out.CredentialWellKnowns[0]["credential_id"])
	require.Equal(
		t,
		map[string]any{"credential_issuer": "issuer-1"},
		out.CredentialWellKnowns[0]["well_known"],
	)
	require.Len(t, out.PresentationResults, 1)
	require.Equal(t, "vp-step", out.PresentationResults[0]["step_id"])
	require.Equal(t, "tenant/use-case-1", out.PresentationResults[0]["use_case_id"])
	result, ok := out.PresentationResults[0]["result"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "jwt", result["format"])
	require.Equal(
		t,
		"haip-vp://?request_uri="+url.QueryEscape(
			server.URL+"/request.jwt",
		)+"&request_uri_method=get",
		result["deeplink_uri"],
	)
	require.Equal(t, map[string]any{"sub": "test"}, result["payload"])
	require.Equal(t, "c2ln", result["signature"])
	require.Equal(t, true, result["signature_present"])
}

func TestPipelineEvidenceExtractionActivityWarnsWhenEmpty(t *testing.T) {
	act := NewPipelineEvidenceExtractionActivity()
	res, err := act.Execute(
		t.Context(),
		workflowengine.ActivityInput{
			Payload: PipelineEvidenceExtractionInput{
				WorkflowDefinition: &pipelineinternal.WorkflowDefinition{Name: "empty"},
				CredimiBaseURL:     "https://credimi.test",
			},
		},
	)
	require.NoError(t, err)

	out, ok := res.Output.(PipelineEvidenceExtractionOutput)
	require.True(t, ok)
	require.Empty(t, out.CredentialWellKnowns)
	require.Empty(t, out.CredentialOffers)
	require.Empty(t, out.PresentationResults)
	require.Contains(
		t,
		out.Warnings,
		"no credential well-knowns or presentation results were extracted",
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
