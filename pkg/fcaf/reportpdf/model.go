// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package reportpdf

import (
	"crypto/sha256"
	"encoding/hex"
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
			document.Unassigned = append(document.Unassigned, image)
			continue
		}
		imagesByEvidence[image.EvidenceKey] = append(imagesByEvidence[image.EvidenceKey], image)
	}

	groupIndex := map[string]int{}
	referencedEvidence := map[string]struct{}{}
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
			entry.Images = append(entry.Images, imagesByEvidence[key]...)
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
		if !referenced {
			document.Unassigned = append(document.Unassigned, imagesByEvidence[key]...)
		}
	}

	return document
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
