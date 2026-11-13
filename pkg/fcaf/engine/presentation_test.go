// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildPresentationExtractsNestedDeeplink(t *testing.T) {
	report := Report{
		Evidence: EvidenceMap{
			"presentation": {
				Value: map[string]any{
					"result": []any{
						map[string]any{"deeplink": "openid4vp://authorize?request_uri=example"},
					},
				},
			},
		},
	}

	presentation := BuildPresentation(report, nil)

	require.Equal(t, "openid4vp://authorize?request_uri=example", presentation.Deeplink)
}

func TestBuildPresentationMapsNonZeroSummaryFiltersInExecutedStatusOrder(t *testing.T) {
	report := Report{
		ExecutedTests: []ExecutedTest{
			{TestID: "a", Status: "passed"},
			{TestID: "b", Status: "passed"},
			{TestID: "c", Status: "passed"},
			{TestID: "d", Status: "passed"},
			{TestID: "e", Status: "failed"},
			{TestID: "f", Status: "failed"},
			{TestID: "g", Status: "failed"},
			{TestID: "h", Status: "blocked"},
			{TestID: "i", Status: "blocked"},
			{TestID: "j", Status: "inconclusive"},
			{TestID: "k", Status: "skipped"},
			{TestID: "l", Status: "not_applicable"},
			{TestID: "m", Status: "not_applicable"},
		},
	}

	presentation := BuildPresentation(report, nil)

	require.Equal(t, []PresentationFilter{
		{Key: "passed", Label: "Passed", Count: 4},
		{Key: "failed", Label: "Failed", Count: 3},
		{Key: "blocked", Label: "Blocked", Count: 2},
		{Key: "skipped", Label: "Skipped", Count: 1},
		{Key: "not_applicable", Label: "Not applicable", Count: 2},
		{Key: "inconclusive", Label: "Inconclusive", Count: 1},
	}, presentation.SummaryFilters)
}

func TestBuildPresentationAssignsVisualsByURLForSharedEvidenceNames(t *testing.T) {
	report := Report{
		ExecutedTests: []ExecutedTest{
			{
				TestID: "cryptography-test",
				Evidence: []ExecutedEvidence{
					{
						Name: "visual_evidence",
						Visual: []string{
							"https://app.test/screens/Cryptography_Result.png?token=first",
						},
					},
				},
			},
			{
				TestID: "trust-test",
				Evidence: []ExecutedEvidence{{
					Name:   "visual_evidence",
					Visual: []string{"https://app.test/screens/Trust-Result.png?token=second"},
				}},
			},
			{
				TestID: "shared-cryptography-test",
				Evidence: []ExecutedEvidence{
					{
						Name: "visual_evidence",
						Visual: []string{
							"https://app.test/screens/Cryptography_Result.png?token=shared",
						},
					},
				},
			},
		},
	}

	presentation := BuildPresentation(report, nil)

	require.Equal(t, []PresentationScreenshot{
		{
			URL:     "https://app.test/screens/Cryptography_Result.png",
			Label:   "Cryptography Result",
			TestIDs: []string{"cryptography-test", "shared-cryptography-test"},
		},
		{
			URL:     "https://app.test/screens/Trust-Result.png",
			Label:   "Trust Result",
			TestIDs: []string{"trust-test"},
		},
	}, presentation.Screenshots)
}

func TestBuildPresentationDoesNotLeakSharedEvidenceNameAcrossTests(t *testing.T) {
	report := Report{Tests: []TestResult{
		{
			ID: "rejection-test",
			Evidence: []EvidenceResult{{
				Name:  "visual_evidence",
				Value: []any{"https://app.test/screens/expect-request-rejected.png"},
			}},
			Assertions: []AssertionResult{
				{ID: "visual", EvidenceKeys: []string{"visual_evidence"}},
			},
		},
		{
			ID: "presentation-test",
			Evidence: []EvidenceResult{{
				Name:  "visual_evidence",
				Value: []any{"https://app.test/screens/engagement-shared.png"},
			}},
			Assertions: []AssertionResult{
				{ID: "visual", EvidenceKeys: []string{"visual_evidence"}},
			},
		},
	}}
	report.PopulateDerivedViews()

	presentation := BuildPresentation(report, nil)

	require.Equal(t, []PresentationScreenshot{
		{
			URL:     "https://app.test/screens/engagement-shared.png",
			Label:   "engagement shared",
			TestIDs: []string{"presentation-test"},
		},
		{
			URL:     "https://app.test/screens/expect-request-rejected.png",
			Label:   "expect request rejected",
			TestIDs: []string{"rejection-test"},
		},
	}, presentation.Screenshots)
}

func TestBuildPresentationIncludesMaestroURLs(t *testing.T) {
	presentation := BuildPresentation(Report{}, []string{
		"https://app.test/screens/Wallet%20Launch_step_1_open.png?token=temporary",
	})

	require.Equal(t, []PresentationScreenshot{{
		URL:   "https://app.test/screens/Wallet%20Launch_step_1_open.png",
		Label: "Wallet Launch step 1 open",
	}}, presentation.Screenshots)
}

func TestBuildPresentationCollectsNestedEvidenceImages(t *testing.T) {
	report := Report{
		Evidence: EvidenceMap{
			"nested": {
				Value: map[string]any{
					"artifacts": []any{
						"https://app.test/screens/evidence.webp?token=evidence",
						"https://app.test/screens/nested-result.jpeg?download=1",
						"https://app.test/screens/not-supported.gif",
					},
				},
			},
		},
	}

	presentation := BuildPresentation(report, []string{
		"https://app.test/screens/evidence.webp?token=maestro",
	})

	require.Equal(t, []PresentationScreenshot{{
		URL:   "https://app.test/screens/evidence.webp",
		Label: "evidence",
	}, {
		URL:   "https://app.test/screens/nested-result.jpeg",
		Label: "nested result",
	}}, presentation.Screenshots)
}

// Two scenarios routinely bind the same evidence name (visual_evidence,
// session) to different sources; each test must keep its own record and its
// own screenshots instead of the last test's.
func TestReportKeepsEvidenceOfTestsSharingAName(t *testing.T) {
	report := Report{Tests: []TestResult{{
		ID:     "first",
		Status: "pass",
		Assertions: []AssertionResult{{
			ID: "visual", Status: "pass", EvidenceKeys: []string{"visual_evidence"},
		}},
		Evidence: []EvidenceResult{{
			Name:  "visual_evidence",
			From:  "pipeline.first.outputs.visual_evidence",
			Value: []any{"https://app.test/screens/first.png"},
		}},
	}, {
		ID:     "second",
		Status: "pass",
		Assertions: []AssertionResult{{
			ID: "visual", Status: "pass", EvidenceKeys: []string{"visual_evidence"},
		}},
		Evidence: []EvidenceResult{{
			Name:  "visual_evidence",
			From:  "pipeline.second.outputs.visual_evidence",
			Value: []any{"https://app.test/screens/second.png"},
		}},
	}}}

	report.PopulateDerivedViews()
	report.AttachPresentation(nil)

	require.Equal(t,
		[]any{"https://app.test/screens/first.png"},
		report.Evidence["pipeline.first.outputs.visual_evidence"].Value,
	)
	require.Equal(t,
		[]any{"https://app.test/screens/second.png"},
		report.Evidence["pipeline.second.outputs.visual_evidence"].Value,
	)
	require.Equal(t,
		[]string{"pipeline.second.outputs.visual_evidence"},
		report.ExecutedTests[1].EvidenceRecordKeys(),
	)
	require.ElementsMatch(t, []PresentationScreenshot{{
		URL:     "https://app.test/screens/first.png",
		Label:   "first",
		TestIDs: []string{"first"},
	}, {
		URL:     "https://app.test/screens/second.png",
		Label:   "second",
		TestIDs: []string{"second"},
	}}, report.Presentation.Screenshots)
}

func TestBuildPresentationKeepsLastScreenshotInBurst(t *testing.T) {
	presentation := BuildPresentation(Report{
		ExecutedTests: []ExecutedTest{{
			TestID: "burst-test",
			Evidence: []ExecutedEvidence{{
				Visual: []string{
					"https://app.test/screens/issuance_screenshot_1_action_tap.yaml1.png",
				},
			}},
		}},
	}, []string{
		"https://app.test/screens/issuance_screenshot_1_action_tap.yaml1.png",
		"https://app.test/screens/issuance_screenshot_2_action_tap.yaml2.png",
	})

	require.Equal(t, []PresentationScreenshot{{
		URL:     "https://app.test/screens/issuance_screenshot_2_action_tap.yaml2.png",
		Label:   "issuance screenshot 2 action tap.yaml2",
		TestIDs: []string{"burst-test"},
	}}, presentation.Screenshots)
}

func TestBuildPresentationDoesNotAssignScreenshotByTestTitleWords(t *testing.T) {
	report := Report{
		Evidence: EvidenceMap{
			"unbound": {
				Value: "https://app.test/screens/email-address-check.png",
			},
		},
		ExecutedTests: []ExecutedTest{{
			TestID: "email-test",
			Title:  "Email address check",
		}},
	}

	presentation := BuildPresentation(report, nil)

	require.Equal(t, []PresentationScreenshot{{
		URL:   "https://app.test/screens/email-address-check.png",
		Label: "email address check",
	}}, presentation.Screenshots)
	require.Empty(t, presentation.Screenshots[0].TestIDs)
}

func TestReportAttachPresentationStoresProjection(t *testing.T) {
	report := Report{ExecutedTests: []ExecutedTest{{TestID: "t1", Status: "passed"}}}

	report.AttachPresentation(nil)

	require.Equal(t, &Presentation{
		SummaryFilters: []PresentationFilter{{
			Key:   "passed",
			Label: "Passed",
			Count: 1,
		}},
	}, report.Presentation)
}
