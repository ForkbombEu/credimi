// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package reportpdf

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
	Runner             RunnerInfo
}

type RunnerInfo struct {
	Name   string
	Type   string
	Serial string
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
	Categories    []Category
	Evidence      []EvidenceEntry
	Unassigned    []ImageAsset
	Warnings      []string
	SelectedCount int
}

type Category struct {
	Code   string
	Name   string
	Color  [3]int
	Groups []TestGroup
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
	images, droppedScreenshots := DeduplicateScreenshots(input.Images)
	if droppedScreenshots > 0 {
		document.Warnings = append(
			document.Warnings,
			fmt.Sprintf("dropped %d duplicate Maestro action screenshots", droppedScreenshots),
		)
	}

	imagesByFilename := make(map[string]ImageAsset, len(images))
	for _, image := range images {
		if _, found := imagesByFilename[image.Filename]; !found {
			imagesByFilename[image.Filename] = image
		}
	}

	categoryGroups := map[string]map[string]*TestGroup{}
	referencedEvidence := map[string]struct{}{}
	assignedImages := map[string]struct{}{}
	presentationScreenshots := input.Report.Presentation != nil &&
		len(input.Report.Presentation.Screenshots) > 0
	for _, execution := range input.Report.ExecutedTests {
		code, subgroup, label := parseTestID(execution.TestID)
		groups, ok := categoryGroups[code]
		if !ok {
			groups = map[string]*TestGroup{}
			categoryGroups[code] = groups
		}
		group, ok := groups[subgroup]
		if !ok {
			group = &TestGroup{Name: label}
			groups[subgroup] = group
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
		}
		if presentationScreenshots {
			for _, screenshot := range input.Report.Presentation.Screenshots {
				if !containsString(screenshot.TestIDs, execution.TestID) {
					continue
				}
				appendReferencedImage(
					&entry.Images,
					assignedImages,
					imagesByFilename,
					screenshot.URL,
				)
			}
		} else {
			for _, item := range execution.Evidence {
				for _, reference := range item.Visual {
					appendReferencedImage(
						&entry.Images,
						assignedImages,
						imagesByFilename,
						reference,
					)
				}
			}
		}
		group.Tests = append(group.Tests, entry)
	}

	for _, code := range categoryOrder {
		groups, ok := categoryGroups[code]
		if !ok {
			continue
		}
		keys := make([]string, 0, len(groups))
		for key := range groups {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		meta := categoryByCode[code]
		category := Category{Code: code, Name: meta.name, Color: meta.color}
		for _, key := range keys {
			category.Groups = append(category.Groups, *groups[key])
		}
		document.Categories = append(document.Categories, category)
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

	// Screenshots not associated with any test render once in the evidence
	// appendix.
	for _, image := range images {
		if _, assigned := assignedImages[image.Filename]; !assigned {
			document.Unassigned = append(document.Unassigned, image)
		}
	}

	return document
}

func appendReferencedImage(
	images *[]ImageAsset,
	assignedImages map[string]struct{},
	imagesByFilename map[string]ImageAsset,
	reference string,
) {
	filename := ReferenceFilename(reference)
	if filename == "" {
		return
	}
	image, found := imagesByFilename[filename]
	if !found {
		return
	}
	appendImageUnique(images, image)
	assignedImages[image.Filename] = struct{}{}
}

func appendImageUnique(images *[]ImageAsset, image ImageAsset) {
	for _, existing := range *images {
		if existing.Filename == image.Filename {
			return
		}
	}
	*images = append(*images, image)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
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
