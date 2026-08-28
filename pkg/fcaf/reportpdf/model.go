// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package reportpdf

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/forkbombeu/credimi/pkg/fcaf/engine"
)

type Metadata struct {
	PipelineName       string
	PipelineIdentifier string
	OrganizationName   string
	WorkflowID         string
	RunID              string
	GeneratedAt        time.Time
	JSONFilename       string
}

type ImageAsset struct {
	EvidenceKey string
	Filename    string
	Data        []byte
}

type Input struct {
	Report      engine.Report
	RawJSON     []byte
	Metadata    Metadata
	Definitions map[string]dsl.TestDefinition
	Sources     map[string]SourceDetails
	Images      []ImageAsset
	Warnings    []string
}

type Document struct {
	Report        engine.Report
	Metadata      Metadata
	JSONSHA256    string
	Groups        []TestGroup
	Evidence      []EvidenceEntry
	Unassigned    []ImageAsset
	Warnings      []string
	SelectedCount int
}

type TestGroup struct {
	Name  string
	Tests []TestEntry
}

type TestEntry struct {
	Execution  engine.ExecutedTest
	Definition dsl.TestDefinition
	Source     SourceDetails
	Evidence   []EvidenceEntry
	Images     []ImageAsset
}

type EvidenceEntry struct {
	Key        string
	Record     engine.EvidenceRecord
	Referenced bool
}

func BuildDocument(input Input) Document {
	document := Document{
		Report:        input.Report,
		Metadata:      input.Metadata,
		Warnings:      append([]string(nil), input.Warnings...),
		SelectedCount: len(input.Report.ExecutedTests),
	}
	if len(input.RawJSON) > 0 {
		digest := sha256.Sum256(input.RawJSON)
		document.JSONSHA256 = hex.EncodeToString(digest[:])
	}

	imagesByEvidence := make(map[string][]ImageAsset)
	for _, image := range input.Images {
		if strings.TrimSpace(image.EvidenceKey) == "" {
			continue
		}
		imagesByEvidence[image.EvidenceKey] = append(imagesByEvidence[image.EvidenceKey], image)
	}

	groupIndex := map[string]int{}
	referencedEvidence := map[string]struct{}{}
	assignedImages := map[string]struct{}{}
	for _, execution := range input.Report.ExecutedTests {
		groupName := testGroupName(execution.TestID)
		index, found := groupIndex[groupName]
		if !found {
			index = len(document.Groups)
			groupIndex[groupName] = index
			document.Groups = append(document.Groups, TestGroup{Name: groupName})
		}

		entry := TestEntry{
			Execution:  execution,
			Definition: input.Definitions[execution.TestID],
			Source:     input.Sources[execution.TestID],
		}
		for _, key := range testEvidenceKeys(execution) {
			record, found := input.Report.Evidence[key]
			if !found {
				document.Warnings = append(
					document.Warnings,
					"test "+execution.TestID+" references missing evidence key "+key,
				)
				continue
			}
			referencedEvidence[key] = struct{}{}
			entry.Evidence = append(entry.Evidence, EvidenceEntry{
				Key:        key,
				Record:     record,
				Referenced: true,
			})
			for _, image := range imagesByEvidence[key] {
				appendImageUnique(&entry.Images, image)
				assignedImages[image.Filename] = struct{}{}
			}
		}
		document.Groups[index].Tests = append(document.Groups[index].Tests, entry)
	}

	keys := make([]string, 0, len(input.Report.Evidence))
	for key := range input.Report.Evidence {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		_, referenced := referencedEvidence[key]
		document.Evidence = append(document.Evidence, EvidenceEntry{
			Key:        key,
			Record:     input.Report.Evidence[key],
			Referenced: referenced,
		})
	}

	// Apply the same filename-based association the webapp report uses so
	// stored screenshots land under the tests whose id or title shares a word
	// with the screenshot name. Exact evidence references are kept in
	// addition: a screenshot referenced by an assertion stays attached to
	// that test even when the name match does not fire.
	allImages := map[string]ImageAsset{}
	for _, image := range input.Images {
		if _, found := allImages[image.Filename]; !found {
			allImages[image.Filename] = image
		}
	}
	imageNames := make([]string, 0, len(allImages))
	for filename := range allImages {
		imageNames = append(imageNames, filename)
	}
	sort.Strings(imageNames)
	for _, filename := range imageNames {
		image := allImages[filename]
		matched := false
		for groupIndex := range document.Groups {
			for testIndex := range document.Groups[groupIndex].Tests {
				if screenshotMatchesTest(filename, document.Groups[groupIndex].Tests[testIndex]) {
					appendImageUnique(
						&document.Groups[groupIndex].Tests[testIndex].Images,
						image,
					)
					matched = true
				}
			}
		}
		if !matched {
			if _, assigned := assignedImages[filename]; !assigned {
				document.Unassigned = append(document.Unassigned, image)
			}
		}
	}

	return document
}

func appendImageUnique(images *[]ImageAsset, image ImageAsset) {
	for _, existing := range *images {
		if existing.Filename == image.Filename {
			return
		}
	}
	*images = append(*images, image)
}

func screenshotLabel(filename string) string {
	if decoded, err := url.PathUnescape(filename); err == nil {
		filename = decoded
	}
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	name = strings.NewReplacer("_", " ", "-", " ").Replace(name)
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}

func screenshotMatchesTest(filename string, test TestEntry) bool {
	label := screenshotLabel(filename)
	if label == "" {
		return false
	}
	searchable := strings.ToLower(test.Execution.TestID + " " + test.Execution.Title)
	for _, word := range strings.Fields(label) {
		if len(word) > 3 && strings.Contains(searchable, word) {
			return true
		}
	}
	return false
}

func testEvidenceKeys(test engine.ExecutedTest) []string {
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
	sort.Strings(keys)
	return keys
}

func testGroupName(testID string) string {
	if index := strings.LastIndex(testID, "__"); index > 0 {
		return testID[:index]
	}
	if strings.TrimSpace(testID) == "" {
		return "Other"
	}
	return testID
}
