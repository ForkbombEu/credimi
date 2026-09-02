// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package reportpdf

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/fcaf/engine"
	"github.com/stretchr/testify/require"
)

func TestParseSourceMarkdown(t *testing.T) {
	source := ParseSourceMarkdown(`# Test

## Objective
Prove wallet behavior.

## References
[OID4VP]

## Profile applicability
EUDI_generic

## EUDI-wallet relevancy
EUDI_required

## Preconditions
Wallet contains PID.

## Test Scenario
1. Open request.
2. Share PID.

## Expected results
1. Presentation succeeds.
`)

	require.Equal(t, "Test", source.Title)
	require.Equal(t, "Prove wallet behavior.", source.Objective)
	require.Equal(t, "EUDI_generic", source.Applicability)
	require.Equal(t, "Wallet contains PID.", source.Preconditions)
	require.Contains(t, source.Scenario, "2. Share PID.")
	require.Equal(t, "1. Presentation succeeds.", source.ExpectedResults)
}

func TestLoadMaterialsFindsCatalogAndSource(t *testing.T) {
	const testID = "WS_RP_IA_MainInteraction__003"

	definitions, sources, warnings := LoadMaterials([]string{testID})
	require.Empty(t, warnings)
	require.Equal(t, testID, definitions[testID].ID)
	require.NotEmpty(t, sources[testID].Objective)
	require.NotEmpty(t, sources[testID].Scenario)
	require.NotEmpty(t, sources[testID].ExpectedResults)
}

func TestBuildDocumentAttachesPerTestVisualEvidence(t *testing.T) {
	report := engine.Report{
		ExecutedTests: []engine.ExecutedTest{
			{
				TestID: "WS_RP_TEST__001",
				Status: "passed",
				Assertions: []engine.ExecutedCheck{
					{ID: "visual", Status: "passed", EvidenceKeys: []string{"visual_evidence"}},
				},
				Evidence: []engine.ExecutedEvidence{{
					Name:       "visual_evidence",
					SourceNode: "pipeline.dcql.cryptography",
					Visual:     []string{"https://app.test/cryptography.png"},
				}},
			},
			{
				TestID: "WS_RP_TEST__002",
				Status: "failed",
				Assertions: []engine.ExecutedCheck{
					{ID: "protocol", Status: "failed", EvidenceKeys: []string{"protocol_evidence"}},
				},
			},
		},
		Evidence: engine.EvidenceMap{
			"visual_evidence": {
				Type:  "json.array",
				Value: []any{"https://app.test/cryptography.png"},
			},
			"protocol_evidence": {Type: "json.object", Value: map[string]any{"status": "failed"}},
		},
	}

	document := BuildDocument(Input{
		Report:  report,
		RawJSON: []byte(`{"status":"failed"}`),
		Images: []ImageAsset{
			{Filename: "cryptography.png", Data: []byte("image")},
			{Filename: "unassigned.png", Data: []byte("image")},
		},
	})

	require.Len(t, document.Categories, 1)
	require.Len(t, document.Categories[0].Groups[0].Tests[0].Images, 1)
	require.Equal(
		t,
		"cryptography.png",
		document.Categories[0].Groups[0].Tests[0].Images[0].Filename,
	)
	// The other test cited no visual evidence, so it gets no screenshot even
	// though the flat report evidence map shares its name.
	require.Empty(t, document.Categories[0].Groups[0].Tests[1].Images)
	require.Len(t, document.Unassigned, 1)
	require.NotEmpty(t, document.JSONSHA256)
}

func TestBuildDocumentFallsBackToWordMatchAssociation(t *testing.T) {
	report := engine.Report{
		ExecutedTests: []engine.ExecutedTest{
			{
				TestID: "WS_RP_DM_Credentialmetadata_Documentnumber__001",
				Status: "passed",
			},
			{
				TestID: "WS_RP_IA_Engagement__001",
				Status: "passed",
			},
		},
	}

	document := BuildDocument(Input{
		Report: report,
		Images: []ImageAsset{
			{Filename: "scenario_obtain_pid_credential_added_x.png", Data: []byte("image")},
			{Filename: "scenario_engagement_complete_y.png", Data: []byte("image")},
		},
	})

	// "credential" matches the Credentialmetadata test id; "engagement"
	// matches the Engagement test id — mirroring the webapp sheet.
	require.Len(t, document.Categories[0].Groups[0].Tests[0].Images, 1)
	require.Equal(
		t,
		"scenario_obtain_pid_credential_added_x.png",
		document.Categories[0].Groups[0].Tests[0].Images[0].Filename,
	)
	require.Len(t, document.Categories[1].Groups[0].Tests[0].Images, 1)
	require.Equal(
		t,
		"scenario_engagement_complete_y.png",
		document.Categories[1].Groups[0].Tests[0].Images[0].Filename,
	)
	require.Empty(t, document.Unassigned)
}

func TestBuildDocumentKeepsScenarioEvidenceSeparate(t *testing.T) {
	report := engine.Report{
		ExecutedTests: []engine.ExecutedTest{
			{
				TestID: "WS_RP_MS_Cryptography__001",
				Status: "passed",
				Evidence: []engine.ExecutedEvidence{{
					Name:       "visual_evidence",
					SourceNode: "pipeline.dcql.cryptography",
					Visual:     []string{"https://app.test/cryptography.png"},
				}},
			},
			{
				TestID: "WS_RP_MS_TrustMechanisms__001",
				Status: "passed",
				Evidence: []engine.ExecutedEvidence{{
					Name:       "visual_evidence",
					SourceNode: "pipeline.dcql.trust-mechanisms",
					Visual:     []string{"https://app.test/trust.png"},
				}},
			},
		},
	}

	document := BuildDocument(Input{
		Report: report,
		Images: []ImageAsset{
			{Filename: "cryptography.png", Data: []byte("image")},
			{Filename: "trust.png", Data: []byte("image")},
		},
	})

	// Both scenarios share the evidence name visual_evidence; each test still
	// receives only its own scenario's screenshot.
	groups := document.Categories[0].Groups
	require.Len(t, groups[0].Tests[0].Images, 1)
	require.Equal(t, "cryptography.png", groups[0].Tests[0].Images[0].Filename)
	require.Len(t, groups[1].Tests[0].Images, 1)
	require.Equal(t, "trust.png", groups[1].Tests[0].Images[0].Filename)
	require.Empty(t, document.Unassigned)
}

func TestReferenceFilenameAndPreparation(t *testing.T) {
	require.Equal(
		t,
		"step.png",
		ReferenceFilename("https://app.test/api/files/pipeline_results/record/step.png?token=x"),
	)
	require.Equal(t, "", ReferenceFilename("https://app.test/"))

	prepared, err := PrepareImage(testPNG(t))
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(prepared, []byte{0xFF, 0xD8}))

	_, err = PrepareImage([]byte("not an image"))
	require.Error(t, err)
}

func TestRenderEmbedsVisualEvidenceImage(t *testing.T) {
	report := engine.Report{
		Suite:  "wallet_solution/relying_party",
		Status: "passed",
		Summary: engine.Summary{
			Pass: 1,
		},
		ExecutedTests: []engine.ExecutedTest{
			{
				TestID: "WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001",
				Status: "passed",
				Assertions: []engine.ExecutedCheck{
					{
						ID:           "email_present",
						Status:       "passed",
						EvidenceKeys: []string{"visual_evidence"},
					},
				},
			},
		},
		Evidence: engine.EvidenceMap{
			"visual_evidence": {Type: "json.array", Value: []any{"https://app.test/visual.png"}},
		},
	}
	imageData, err := PrepareImage(testPNG(t))
	require.NoError(t, err)
	document := BuildDocument(Input{
		Report:  report,
		RawJSON: []byte(`{"status":"passed"}`),
		Images: []ImageAsset{
			{EvidenceKey: "visual_evidence", Filename: "visual.png", Data: imageData},
		},
		Metadata: Metadata{
			PipelineName: "FCAF complete validation",
			WorkflowID:   "workflow-1",
			RunID:        "run-1",
			GeneratedAt:  time.Date(2026, time.August, 28, 0, 0, 0, 0, time.UTC),
			JSONFilename: "fcaf-assessment.json",
		},
	})

	data, err := Render(context.Background(), document)
	require.NoError(t, err)
	require.True(t, bytes.Contains(data, []byte("/Subtype /Image")))
}

func TestRenderProducesMultipagePDF(t *testing.T) {
	tests := make([]engine.ExecutedTest, 0, 18)
	for index := range 18 {
		tests = append(tests, engine.ExecutedTest{
			TestID: "WS_RP_LONG__" + string(rune('A'+index)),
			Title:  "Long FCAF test explanation",
			Status: "passed",
			Assertions: []engine.ExecutedCheck{
				{ID: "assertion", Status: "passed", Message: "Evidence satisfies expected result."},
			},
		})
	}
	report := engine.Report{
		Suite:         "wallet_solution/relying_party",
		Status:        "passed",
		ExecutedTests: tests,
		Summary:       engine.Summary{Pass: len(tests)},
	}
	document := BuildDocument(Input{
		Report:  report,
		RawJSON: []byte(`{"status":"passed"}`),
		Metadata: Metadata{
			PipelineName:     "Complete FCAF validation",
			OrganizationName: "Test organization",
			WorkflowID:       "workflow-1",
			RunID:            "run-1",
			GeneratedAt:      time.Date(2026, time.August, 28, 0, 0, 0, 0, time.UTC),
			JSONFilename:     "fcaf-assessment.json",
		},
	})

	data, err := Render(context.Background(), document)
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(data, []byte("%PDF-")))
	require.Greater(t, bytes.Count(data, []byte("/Type /Page")), 2)
}

func TestDeduplicateScreenshotsKeepsLastPerBurst(t *testing.T) {
	images := []ImageAsset{
		{
			Filename: "pid_mdoc_8f915369_obtain_pid_mdoc_screenshot_1788208561028_action_a1.yaml1.png",
		},
		{Filename: "pid_mdoc_8f915369_obtain_pid_mdoc_credential_added_x1.png"},
		{
			Filename: "pid_mdoc_8f915369_obtain_pid_mdoc_screenshot_1788208562014_action_a2.yaml1.png",
		},
		{
			Filename: "engagement_4bb0f83a_obtain_pid_sdjwt_screenshot_1788207713069_action_b1.yaml2.png",
		},
		{
			Filename: "pid_mdoc_8f915369_obtain_pid_mdoc_screenshot_1788208562614_action_a3.yaml1.png",
		},
		{
			Filename: "engagement_4bb0f83a_obtain_pid_sdjwt_screenshot_1788207713943_action_b2.yaml2.png",
		},
	}

	kept, dropped := DeduplicateScreenshots(images)

	require.Equal(t, 3, dropped)
	require.Len(t, kept, 3)
	require.Equal(
		t,
		"pid_mdoc_8f915369_obtain_pid_mdoc_credential_added_x1.png",
		kept[0].Filename,
	)
	require.Equal(
		t,
		"pid_mdoc_8f915369_obtain_pid_mdoc_screenshot_1788208562614_action_a3.yaml1.png",
		kept[1].Filename,
	)
	require.Equal(
		t,
		"engagement_4bb0f83a_obtain_pid_sdjwt_screenshot_1788207713943_action_b2.yaml2.png",
		kept[2].Filename,
	)
}

func TestDeduplicateScreenshotsLeavesNonBurstImagesAlone(t *testing.T) {
	images := []ImageAsset{
		{Filename: "onboard_reference_wallet_fcaf_onboarding_complete_y.png"},
		{Filename: "dcql_cryptography_1a488a33_exercise_wallet_cryptography_x.png"},
	}

	kept, dropped := DeduplicateScreenshots(images)

	require.Equal(t, 0, dropped)
	require.Len(t, kept, 2)
}

func testPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			img.Set(x, y, color.RGBA{R: 49, G: 4, B: 255, A: 255})
		}
	}
	var output bytes.Buffer
	require.NoError(t, png.Encode(&output, img))
	return output.Bytes()
}

func TestDeduplicateScreenshotsKeepsLastPerCloudBurst(t *testing.T) {
	images := []ImageAsset{
		{Filename: "engagement_haip_vp_4bb0f83a_obtain_pid_sdjwt_step_004_tap_on_element_eudi_wallet_a.png"},
		{Filename: "engagement_haip_vp_4bb0f83a_obtain_pid_sdjwt_step_005_tap_on_element_just_once_b.png"},
		{Filename: "engagement_haip_vp_4bb0f83a_obtain_pid_sdjwt_step_006_tap_on_element_always_c.png"},
		{Filename: "engagement_haip_vp_4bb0f83a_obtain_pid_sdjwt_credential_added_d.png"},
		{Filename: "engagement_haip_vp_4bb0f83a_invoke_wallet_with_haip_vp_fcaf_engagement_haip_vp_invoked_e.png"},
	}

	kept, dropped := DeduplicateScreenshots(images)

	require.Equal(t, 2, dropped)
	require.Len(t, kept, 3)
	require.Equal(
		t,
		"engagement_haip_vp_4bb0f83a_obtain_pid_sdjwt_step_006_tap_on_element_always_c.png",
		kept[0].Filename,
	)
	require.Equal(
		t,
		"engagement_haip_vp_4bb0f83a_obtain_pid_sdjwt_credential_added_d.png",
		kept[1].Filename,
	)
	require.Equal(
		t,
		"engagement_haip_vp_4bb0f83a_invoke_wallet_with_haip_vp_fcaf_engagement_haip_vp_invoked_e.png",
		kept[2].Filename,
	)
}
