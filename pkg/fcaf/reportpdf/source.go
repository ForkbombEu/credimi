// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package reportpdf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/forkbombeu/credimi/pkg/fcaf/catalog"
	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
)

const (
	defaultCatalogRoot = "config_templates/fcaf/wallet_solution/relying_party"
	defaultSourceRoot  = "config_templates/fcaf_sources/wallet_solution/relying_party/implementation"
)

type SourceDetails struct {
	Title           string
	Objective       string
	References      string
	Applicability   string
	WalletRelevancy string
	Preconditions   string
	Scenario        string
	ExpectedResults string
}

func LoadMaterials(testIDs []string) (
	map[string]dsl.TestDefinition,
	map[string]SourceDetails,
	[]string,
) {
	definitions := map[string]dsl.TestDefinition{}
	sources := map[string]SourceDetails{}
	warnings := []string{}

	catalogRoot, found := resolveProjectPath(defaultCatalogRoot)
	if !found {
		warnings = append(warnings, "FCAF catalog root was not found")
	} else {
		loaded, err := catalog.LoadTests(filepath.Join(catalogRoot, "tests"))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("load FCAF catalog: %v", err))
		} else {
			for _, id := range testIDs {
				definition, ok := loaded[id]
				if !ok {
					warnings = append(warnings, "FCAF catalog definition was not found for "+id)
					continue
				}
				definitions[id] = definition
			}
		}
	}

	sourceRoot, found := resolveProjectPath(defaultSourceRoot)
	if !found {
		warnings = append(warnings, "FCAF source root was not found")
		return definitions, sources, warnings
	}
	for _, id := range testIDs {
		data, err := os.ReadFile(filepath.Join(sourceRoot, filepath.Base(id)+".md"))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("load FCAF source for %s: %v", id, err))
			continue
		}
		sources[id] = ParseSourceMarkdown(string(data))
	}

	return definitions, sources, warnings
}

func ParseSourceMarkdown(markdown string) SourceDetails {
	markdown = strings.ReplaceAll(markdown, "\r\n", "\n")
	sections := map[string][]string{}
	current := "title"
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			current = strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			continue
		}
		if strings.HasPrefix(trimmed, "# ") && current == "title" {
			sections[current] = append(
				sections[current],
				strings.TrimSpace(strings.TrimPrefix(trimmed, "# ")),
			)
			continue
		}
		sections[current] = append(sections[current], line)
	}

	return SourceDetails{
		Title:           sectionText(sections, "title"),
		Objective:       sectionText(sections, "Objective"),
		References:      sectionText(sections, "References"),
		Applicability:   sectionText(sections, "Profile applicability"),
		WalletRelevancy: sectionText(sections, "EUDI-wallet relevancy"),
		Preconditions:   sectionText(sections, "Preconditions"),
		Scenario:        sectionText(sections, "Test Scenario"),
		ExpectedResults: sectionText(sections, "Expected results"),
	}
}

func sectionText(sections map[string][]string, key string) string {
	lines := sections[key]
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func resolveProjectPath(relative string) (string, bool) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return relative, false
	}
	for directory := workingDirectory; ; directory = filepath.Dir(directory) {
		candidate := filepath.Join(directory, relative)
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate, true
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
	}
	return relative, false
}
