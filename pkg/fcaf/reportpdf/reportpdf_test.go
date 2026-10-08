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
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/forkbombeu/credimi/pkg/fcaf/engine"
	"github.com/go-pdf/fpdf"
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
		Presentation: &engine.Presentation{},
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

func TestBuildDocumentUsesPresentationTestIDsForScreenshotAssignment(t *testing.T) {
	report := engine.Report{
		ExecutedTests: []engine.ExecutedTest{
			{
				TestID: "WS_RP_TEST__001",
				Title:  "Credential metadata",
				Status: "passed",
				Evidence: []engine.ExecutedEvidence{{
					Name:   "visual_evidence",
					Visual: []string{"https://app.test/evidence-only.png"},
				}},
			},
			{
				TestID: "WS_RP_TEST__002",
				Status: "passed",
			},
		},
		Presentation: &engine.Presentation{
			Screenshots: []engine.PresentationScreenshot{
				{
					URL:     "https://app.test/presentation%20assigned.png?token=x",
					TestIDs: []string{"WS_RP_TEST__002"},
				},
				{
					URL:   "https://app.test/credential-metadata.png",
					Label: "credential metadata",
				},
			},
		},
	}

	document := BuildDocument(Input{
		Report: report,
		Images: []ImageAsset{
			{Filename: "presentation assigned.png", Data: []byte("image")},
			{Filename: "credential-metadata.png", Data: []byte("image")},
			{Filename: "evidence-only.png", Data: []byte("image")},
		},
	})

	tests := document.Categories[0].Groups[0].Tests
	require.Empty(t, tests[0].Images)
	require.Len(t, tests[1].Images, 1)
	require.Equal(
		t,
		"presentation assigned.png",
		tests[1].Images[0].Filename,
	)
	require.ElementsMatch(
		t,
		[]ImageAsset{
			{Filename: "credential-metadata.png", Data: []byte("image")},
			{Filename: "evidence-only.png", Data: []byte("image")},
		},
		document.Unassigned,
	)
}

func TestBuildDocumentAssignsPresentationScreenshotBuiltFromEvidenceKey(t *testing.T) {
	report := engine.Report{
		Evidence: engine.EvidenceMap{
			"visual_evidence": {
				Value: map[string]any{
					"artifacts": []any{"https://app.test/evidence-only.png?token=temporary"},
				},
			},
		},
		ExecutedTests: []engine.ExecutedTest{{
			TestID: "WS_RP_TEST__001",
			Status: "passed",
			Assertions: []engine.ExecutedCheck{{
				ID:           "visual",
				Status:       "passed",
				EvidenceKeys: []string{"visual_evidence"},
			}},
		}},
	}
	report.AttachPresentation(nil)

	document := BuildDocument(Input{
		Report: report,
		Images: []ImageAsset{
			{Filename: "evidence-only.png", Data: []byte("image")},
		},
	})

	tests := document.Categories[0].Groups[0].Tests
	require.Len(t, tests[0].Images, 1)
	require.Equal(t, "evidence-only.png", tests[0].Images[0].Filename)
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
	return testPNGWithColor(t, color.RGBA{R: 49, G: 4, B: 255, A: 255})
}

func testPNGWithColor(t *testing.T, fill color.RGBA) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			img.Set(x, y, fill)
		}
	}
	var output bytes.Buffer
	require.NoError(t, png.Encode(&output, img))
	return output.Bytes()
}

func TestDeduplicateScreenshotsKeepsLastPerCloudBurst(t *testing.T) {
	images := []ImageAsset{
		{
			Filename: "engagement_haip_vp_4bb0f83a_obtain_pid_sdjwt_step_004_tap_on_element_eudi_wallet_a.png",
		},
		{
			Filename: "engagement_haip_vp_4bb0f83a_obtain_pid_sdjwt_step_005_tap_on_element_just_once_b.png",
		},
		{
			Filename: "engagement_haip_vp_4bb0f83a_obtain_pid_sdjwt_step_006_tap_on_element_always_c.png",
		},
		{Filename: "engagement_haip_vp_4bb0f83a_obtain_pid_sdjwt_credential_added_d.png"},
		{
			Filename: "engagement_haip_vp_4bb0f83a_invoke_wallet_with_haip_vp_fcaf_engagement_haip_vp_invoked_e.png",
		},
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

func TestRenderEmbedsVisualEvidenceOfEveryTest(t *testing.T) {
	firstTestImages := []string{"shared.png", "a.png", "b.png", "c.png", "d.png", "empty.png"}
	images := make([]ImageAsset, 0, len(firstTestImages))
	for index, filename := range firstTestImages {
		// fpdf embeds byte-identical images once, so every screenshot gets its own color.
		red := []uint8{0, 40, 80, 120, 160, 200}[index]
		imageData, err := PrepareImage(testPNGWithColor(t, color.RGBA{
			R: red, G: 120, B: 200, A: 255,
		}))
		require.NoError(t, err)
		asset := ImageAsset{Filename: filename, Data: imageData}
		if filename == "empty.png" {
			asset.Data = nil
		}
		images = append(images, asset)
	}
	report := engine.Report{
		Status: "failed",
		ExecutedTests: []engine.ExecutedTest{
			{
				TestID:   "WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001",
				Status:   "failed",
				Outcome:  engine.TestOutcome{Status: "failed", Reason: "Email claim missing."},
				Evidence: []engine.ExecutedEvidence{{Name: "screens", Visual: firstTestImages}},
			},
			{
				TestID: "WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_002",
				Status: "passed",
				Evidence: []engine.ExecutedEvidence{
					{Name: "screens", Visual: []string{"shared.png"}},
				},
			},
		},
	}
	metadata := Metadata{GeneratedAt: time.Date(2026, time.August, 28, 0, 0, 0, 0, time.UTC)}

	document := BuildDocument(Input{Report: report, Images: images, Metadata: metadata})
	require.Empty(t, document.Unassigned)
	require.Len(t, document.Categories, 1)
	tests := document.Categories[0].Groups[0].Tests
	require.Len(t, tests[0].Images, len(firstTestImages))
	require.Equal(t, []ImageAsset{images[0]}, tests[1].Images)
	withImages, err := Render(context.Background(), document)
	require.NoError(t, err)
	withoutImages, err := Render(
		context.Background(),
		BuildDocument(Input{Report: report, Metadata: metadata}),
	)
	require.NoError(t, err)

	// Five non-empty screenshots across a two-row grid (the shared one is registered
	// once); the empty one has nothing to embed.
	require.Equal(
		t,
		bytes.Count(withoutImages, []byte("/Subtype /Image"))+5,
		bytes.Count(withImages, []byte("/Subtype /Image")),
	)
	require.True(t, pdfHasBookmark(withImages, report.ExecutedTests[0].TestID))
	require.True(t, pdfHasBookmark(withImages, report.ExecutedTests[1].TestID))
}

func TestRenderAddsWarningsSectionOnlyWhenNeeded(t *testing.T) {
	report := engine.Report{
		Status: "passed",
		ExecutedTests: []engine.ExecutedTest{
			{TestID: "WS_RP_IA_MainInteraction__003", Status: "passed"},
		},
	}

	clean, err := Render(context.Background(), BuildDocument(Input{Report: report}))
	require.NoError(t, err)
	require.False(t, pdfHasBookmark(clean, "Report warnings"))
	require.True(t, pdfHasBookmark(clean, "WS_RP_IA_MainInteraction__003"))

	warned, err := Render(context.Background(), BuildDocument(Input{
		Report: report,
		Warnings: []string{
			"visual evidence z.png was not stored",
			"load pipeline metadata: missing",
		},
	}))
	require.NoError(t, err)
	require.True(t, pdfHasBookmark(warned, "Report warnings"))
	require.Greater(
		t,
		bytes.Count(warned, []byte("/Type /Page\n")),
		bytes.Count(clean, []byte("/Type /Page\n")),
		"warnings get their own page",
	)
}

func TestRenderStopsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	data, err := Render(ctx, BuildDocument(Input{Report: engine.Report{
		ExecutedTests: []engine.ExecutedTest{{TestID: "WS_RP_IA_MainInteraction__003"}},
	}}))

	require.ErrorIs(t, err, context.Canceled)
	require.ErrorContains(t, err, "generate FCAF PDF")
	require.Nil(t, data)
}

func TestStatusLabelAndColor(t *testing.T) {
	green := [3]int{35, 126, 74}
	red := [3]int{181, 48, 48}
	amber := [3]int{164, 103, 16}
	grey := [3]int{99, 99, 109}
	purple := [3]int{74, 55, 168}

	tests := []struct {
		status    string
		wantLabel string
		wantColor [3]int
	}{
		{status: "pass", wantLabel: "Passed", wantColor: green},
		{status: " PASSED ", wantLabel: "Passed", wantColor: green},
		{status: "fail", wantLabel: "Failed", wantColor: red},
		{status: "Failed", wantLabel: "Failed", wantColor: red},
		{status: "error", wantLabel: "Error", wantColor: red},
		{status: "blocked", wantLabel: "Blocked", wantColor: amber},
		{status: "inconclusive", wantLabel: "Inconclusive", wantColor: amber},
		{status: "skipped", wantLabel: "Skipped", wantColor: grey},
		{status: "not_applicable", wantLabel: "Not applicable", wantColor: grey},
		{status: "not applicable", wantLabel: "Not applicable", wantColor: grey},
		{status: "", wantLabel: "Unknown", wantColor: purple},
		{status: "pending", wantLabel: "Pending", wantColor: purple},
	}

	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			label := statusLabel(tc.status)
			require.Equal(t, tc.wantLabel, label)
			red, green, blue := statusColor(label)
			require.Equal(t, tc.wantColor, [3]int{red, green, blue})
		})
	}
}

func TestWrapTokenBreaksLongIdentifiersAtUnderscores(t *testing.T) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("Inter", "", interRegular)
	pdf.SetFont("Inter", "", 9)
	r := &renderer{pdf: pdf}
	const width = 40.0

	require.Equal(t, "", r.wrapToken("   ", width))
	require.Equal(t, "short_id", r.wrapToken("  short_id  ", width))

	const unbreakable = "averyveryverylongidentifierwithoutanyunderscoreseparators"
	require.Equal(t, unbreakable, r.wrapToken(unbreakable, width))

	const testID = "WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001"
	wrapped := r.wrapToken(testID, width)
	lines := strings.Split(wrapped, "\n")
	require.Greater(t, len(lines), 1)
	require.Equal(t, testID, strings.Join(lines, ""), "wrapping only inserts line breaks")
	for _, line := range lines[:len(lines)-1] {
		require.True(
			t,
			strings.HasSuffix(line, "_"),
			"line %q must break after an underscore",
			line,
		)
	}
	for _, line := range lines {
		if strings.Count(line, "_") > 1 {
			require.LessOrEqual(t, pdf.GetStringWidth(line), width, "line %q overflows", line)
		}
	}
}

// pdfHasBookmark reports whether the PDF outline contains title. Bookmarks are
// written as UTF-16BE strings because the report uses UTF-8 fonts.
func pdfHasBookmark(data []byte, title string) bool {
	encoded := make([]byte, 0, 2*len(title))
	for _, unit := range utf16.Encode([]rune(title)) {
		encoded = append(encoded, byte(unit>>8), byte(unit))
	}
	return bytes.Contains(data, encoded)
}
