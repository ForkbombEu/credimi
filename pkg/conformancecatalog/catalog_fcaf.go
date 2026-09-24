// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type fcafTestFileMeta struct {
	ID       string `yaml:"id"`
	Title    string `yaml:"title"`
	Protocol string `yaml:"protocol"`
	Provider string `yaml:"provider"`
	Suite    struct {
		SUT  string `yaml:"sut"`
		Role string `yaml:"role"`
	} `yaml:"suite"`
}

// hasFCAFTestsDir reports whether this suite stores FCAF definitions under tests/.
// Classic suites keep check YAML at the suite root; FCAF uses tests/*.yaml.
func hasFCAFTestsDir(standardUID, suitePath string) bool {
	if standardUID != "fcaf" {
		return false
	}
	info, err := os.Stat(filepath.Join(suitePath, "tests"))
	return err == nil && info.IsDir()
}

// loadFCAFSuiteTests indexes FCAF definitions from suite/tests/*.yaml.
// Path identity is fcaf/<version>/<suite>/<test_id> so nest/pickers stay stable;
// the final path segment is the FCAF test id used by fcaf-validation.test_ids.
func loadFCAFSuiteTests(
	suitePath, standardUID, versionUID, suiteUID string,
	visibleIn []string,
	suiteFacets facetFields,
) ([]Check, error) {
	testsDir := filepath.Join(suitePath, "tests")
	entries, err := os.ReadDir(testsDir)
	if err != nil {
		return nil, fmt.Errorf("read FCAF tests dir %s: %w", testsDir, err)
	}

	var checks []Check
	for _, f := range entries {
		if f.IsDir() {
			continue
		}
		fileName := f.Name()
		ext := strings.ToLower(filepath.Ext(fileName))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		filePath := filepath.Join(testsDir, fileName)
		meta := fcafTestFileMeta{}
		if err := readRequiredYAML(filePath, &meta); err != nil {
			return nil, err
		}
		testID := strings.TrimSpace(meta.ID)
		if testID == "" {
			testID = strings.TrimSuffix(fileName, filepath.Ext(fileName))
		}
		if testID == "" {
			continue
		}

		title := strings.TrimSpace(meta.Title)
		if title == "" {
			title = testID
		}

		facets := resolveFacets(standardUID, suiteUID, suiteFacets, facetFields{
			Protocol: meta.Protocol,
			SUT:      meta.Suite.SUT,
			Role:     meta.Suite.Role,
			Provider: meta.Provider,
		})

		checks = append(checks, newSuiteCheck(
			standardUID, versionUID, suiteUID, testID, fileName,
			title, visibleIn, facets,
		))
	}
	return checks, nil
}
