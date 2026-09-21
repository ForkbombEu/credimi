// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
)

var presentationBurstScreenshot = regexp.MustCompile(
	`^(.+)(?:_screenshot_\d+_action_[A-Za-z0-9_]+\.yaml\d+|_step_\d+_[A-Za-z0-9_]+)\.png$`,
)

// Presentation is the display projection nested inside an FCAF assessment report.
type Presentation struct {
	Deeplink       string                   `json:"deeplink,omitempty"`
	Screenshots    []PresentationScreenshot `json:"screenshots,omitempty"`
	SummaryFilters []PresentationFilter     `json:"summary_filters,omitempty"`
}

// PresentationScreenshot identifies an image and the tests that use it as visual evidence.
type PresentationScreenshot struct {
	URL     string   `json:"url"`
	Label   string   `json:"label"`
	TestIDs []string `json:"test_ids,omitempty"`
}

// PresentationFilter is a non-empty summary count aligned to an executed status.
type PresentationFilter struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

// BuildPresentation builds the display projection for an FCAF assessment report.
func BuildPresentation(report Report, maestroURLs []string) Presentation {
	return Presentation{
		Deeplink:       evidenceMapDeeplink(report.Evidence),
		Screenshots:    presentationScreenshots(report, maestroURLs),
		SummaryFilters: presentationSummaryFilters(report),
	}
}

// AttachPresentation builds and stores the report's display projection.
func (r *Report) AttachPresentation(maestroURLs []string) {
	if r == nil {
		return
	}
	presentation := BuildPresentation(*r, maestroURLs)
	r.Presentation = &presentation
}

func presentationScreenshots(report Report, maestroURLs []string) []PresentationScreenshot {
	visualTestIDs := make(map[string][]string)
	urls := make([]string, 0)
	seen := make(map[string]struct{})
	for _, reference := range maestroURLs {
		normalized := normalizePresentationURL(reference)
		if normalized == "" {
			continue
		}
		if _, found := seen[normalized]; found {
			continue
		}
		seen[normalized] = struct{}{}
		urls = append(urls, normalized)
	}
	evidenceKeys := make([]string, 0, len(report.Evidence))
	for key := range report.Evidence {
		evidenceKeys = append(evidenceKeys, key)
	}
	sort.Strings(evidenceKeys)
	for _, key := range evidenceKeys {
		for _, reference := range ImageReferenceURLs(report.Evidence[key].Value) {
			if !isPresentationEvidenceImage(reference) {
				continue
			}
			appendPresentationReference(&urls, seen, reference)
		}
	}
	for _, test := range report.ExecutedTests {
		for _, key := range presentationEvidenceKeys(test) {
			record, found := report.Evidence[key]
			if !found {
				continue
			}
			for _, reference := range ImageReferenceURLs(record.Value) {
				if !isPresentationEvidenceImage(reference) {
					continue
				}
				normalized := appendPresentationReference(&urls, seen, reference)
				if normalized == "" {
					continue
				}
				visualTestIDs[normalized] = appendUniqueString(visualTestIDs[normalized], test.TestID)
			}
		}
		for _, evidence := range test.Evidence {
			for _, reference := range evidence.Visual {
				normalized := appendPresentationReference(&urls, seen, reference)
				if normalized == "" {
					continue
				}
				visualTestIDs[normalized] = appendUniqueString(
					visualTestIDs[normalized],
					test.TestID,
				)
			}
		}
	}

	if len(urls) == 0 {
		return nil
	}
	screenshots := make([]PresentationScreenshot, 0, len(urls))
	for _, reference := range urls {
		screenshots = append(screenshots, PresentationScreenshot{
			URL:     reference,
			Label:   presentationScreenshotLabel(reference),
			TestIDs: visualTestIDs[reference],
		})
	}
	return dedupePresentationBursts(screenshots)
}

func appendPresentationReference(urls *[]string, seen map[string]struct{}, reference string) string {
	normalized := normalizePresentationURL(reference)
	if normalized == "" {
		return ""
	}
	if _, found := seen[normalized]; !found {
		seen[normalized] = struct{}{}
		*urls = append(*urls, normalized)
	}
	return normalized
}

func dedupePresentationBursts(screenshots []PresentationScreenshot) []PresentationScreenshot {
	lastOfBurst := make(map[string]int)
	burstPrefix := make([]string, len(screenshots))
	for index, screenshot := range screenshots {
		filename := path.Base(screenshot.URL)
		if decoded, err := url.PathUnescape(filename); err == nil {
			filename = decoded
		}
		match := presentationBurstScreenshot.FindStringSubmatch(filename)
		if match == nil {
			continue
		}
		burstPrefix[index] = match[1]
		lastOfBurst[match[1]] = index
	}

	merged := make([]PresentationScreenshot, len(screenshots))
	copy(merged, screenshots)
	for index, screenshot := range screenshots {
		prefix := burstPrefix[index]
		if prefix == "" {
			continue
		}
		keepAt := lastOfBurst[prefix]
		if keepAt == index {
			continue
		}
		for _, testID := range screenshot.TestIDs {
			merged[keepAt].TestIDs = appendUniqueString(merged[keepAt].TestIDs, testID)
		}
	}

	kept := make([]PresentationScreenshot, 0, len(merged))
	for index, screenshot := range merged {
		if prefix := burstPrefix[index]; prefix != "" && lastOfBurst[prefix] != index {
			continue
		}
		kept = append(kept, screenshot)
	}
	return kept
}

func isPresentationEvidenceImage(reference string) bool {
	parsed, err := url.Parse(strings.TrimSpace(reference))
	if err != nil {
		return false
	}
	switch strings.ToLower(path.Ext(parsed.Path)) {
	case ".jpeg", ".jpg", ".png", ".webp":
		return true
	default:
		return false
	}
}

func normalizePresentationURL(reference string) string {
	reference = strings.TrimSpace(reference)
	if query := strings.IndexByte(reference, '?'); query >= 0 {
		reference = reference[:query]
	}
	return reference
}

func presentationScreenshotLabel(reference string) string {
	filename := path.Base(reference)
	if decoded, err := url.PathUnescape(filename); err == nil {
		filename = decoded
	}
	filename = strings.TrimSuffix(filename, path.Ext(filename))
	filename = strings.NewReplacer("_", " ", "-", " ").Replace(filename)
	return strings.Join(strings.Fields(filename), " ")
}

func appendUniqueString(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func presentationEvidenceKeys(test ExecutedTest) []string {
	seen := map[string]struct{}{}
	keys := make([]string, 0)
	for _, assertion := range test.Assertions {
		for _, key := range assertion.EvidenceKeys {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			if _, found := seen[key]; found {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, key)
		}
	}
	for _, evidence := range test.Evidence {
		key := strings.TrimSpace(evidence.Name)
		if key == "" {
			continue
		}
		if _, found := seen[key]; found {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func presentationSummaryFilters(report Report) []PresentationFilter {
	counts := map[string]int{}
	for _, test := range report.ExecutedTests {
		status := strings.TrimSpace(test.Status)
		if status == "" {
			continue
		}
		counts[status]++
	}
	candidates := []PresentationFilter{
		{Key: "passed", Label: "Passed"},
		{Key: "failed", Label: "Failed"},
		{Key: "blocked", Label: "Blocked"},
		{Key: "skipped", Label: "Skipped"},
		{Key: "inconclusive", Label: "Inconclusive"},
	}
	filters := make([]PresentationFilter, 0, len(candidates))
	for _, filter := range candidates {
		filter.Count = counts[filter.Key]
		if filter.Count > 0 {
			filters = append(filters, filter)
		}
	}
	return filters
}

func evidenceMapDeeplink(evidence EvidenceMap) string {
	keys := make([]string, 0, len(evidence))
	for key := range evidence {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if deeplink := nestedDeeplink(evidence[key].Value); deeplink != "" {
			return deeplink
		}
	}
	return ""
}

func nestedDeeplink(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			nested := typed[key]
			if key == "deeplink" {
				if deeplink, ok := nested.(string); ok {
					return deeplink
				}
			}
			if deeplink := nestedDeeplink(nested); deeplink != "" {
				return deeplink
			}
		}
	case []any:
		for _, nested := range typed {
			if deeplink := nestedDeeplink(nested); deeplink != "" {
				return deeplink
			}
		}
	}
	return ""
}
