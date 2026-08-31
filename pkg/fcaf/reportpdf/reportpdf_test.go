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

func TestBuildDocumentAssociatesOnlyExactEvidenceImages(t *testing.T) {
	report := engine.Report{
		ExecutedTests: []engine.ExecutedTest{
			{
				TestID: "WS_RP_TEST__001",
				Status: "passed",
				Assertions: []engine.ExecutedCheck{
					{ID: "visual", Status: "passed", EvidenceKeys: []string{"visual_evidence"}},
				},
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
			"visual_evidence":   {Type: "json.array", Value: []any{"https://app.test/visual.png"}},
			"protocol_evidence": {Type: "json.object", Value: map[string]any{"status": "failed"}},
		},
	}

	document := BuildDocument(Input{
		Report:  report,
		RawJSON: []byte(`{"status":"failed"}`),
		Images: []ImageAsset{
			{EvidenceKey: "visual_evidence", Filename: "visual.png", Data: []byte("image")},
			{Filename: "unassigned.png", Data: []byte("image")},
		},
	})

	require.Len(t, document.Categories, 1)
	require.Len(t, document.Categories[0].Groups[0].Tests[0].Images, 1)
	require.Empty(t, document.Categories[0].Groups[0].Tests[1].Images)
	require.Len(t, document.Unassigned, 1)
	require.NotEmpty(t, document.JSONSHA256)
}

func TestBuildDocumentFallsBackToFilenameAssociation(t *testing.T) {
	report := engine.Report{
		ExecutedTests: []engine.ExecutedTest{
			{
				TestID: "WS_RP_DM_Credentialmetadata_Documentnumber__001",
				Status: "failed",
			},
			{
				TestID: "WS_RP_DM_AddressData_Emailaddress__001",
				Status: "failed",
			},
		},
		Evidence: engine.EvidenceMap{
			// visual_evidence resolved to null because the producing step failed.
			"visual_evidence": {Type: "json.array", Value: nil},
		},
	}

	document := BuildDocument(Input{
		Report: report,
		Images: []ImageAsset{
			{Filename: "getcredential_generic_credential_without_authentication_0004_credential_added_x.png", Data: []byte("image")},
			{Filename: "onboarding_1_fcaf_onboarding_complete_y.png", Data: []byte("image")},
		},
	})

	require.Len(t, document.Categories, 1)
	require.Len(t, document.Categories[0].Groups, 2)
	// Subgroups sort alphabetically: AddressData before Credentialmetadata.
	require.Empty(t, document.Categories[0].Groups[0].Tests[0].Images)
	// "credential" word overlaps the Credentialmetadata test id.
	require.Len(t, document.Categories[0].Groups[1].Tests[0].Images, 1)
	require.Len(t, document.Unassigned, 1)
}

func TestImageReferencesAndPreparation(t *testing.T) {
	report := engine.Report{Evidence: engine.EvidenceMap{
		"visual": {
			Value: map[string]any{
				"screenshots": []any{
					"https://app.test/api/files/pipeline_results/record/step.png?token=x",
					"https://app.test/result.json",
				},
			},
		},
	}}

	references := ImageReferences(report)
	require.Equal(
		t,
		[]string{"https://app.test/api/files/pipeline_results/record/step.png?token=x"},
		references["visual"],
	)
	require.Equal(t, "step.png", ReferenceFilename(references["visual"][0]))

	prepared, err := PrepareImage(testPNG(t))
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(prepared, []byte("\x89PNG")))

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
					{ID: "email_present", Status: "passed", EvidenceKeys: []string{"visual_evidence"}},
				},
			},
		},
		Evidence: engine.EvidenceMap{
			"visual_evidence": {Type: "json.array", Value: []any{"https://app.test/visual.png"}},
		},
	}
	imageData := testPNG(t)
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
