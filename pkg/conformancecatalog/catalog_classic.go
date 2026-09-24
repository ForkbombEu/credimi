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

// loadClassicSuiteChecks indexes check YAML at the suite root (not under tests/).
func loadClassicSuiteChecks(
	suitePath, standardUID, versionUID, suiteUID string,
	visibleIn []string,
	suiteFacets facetFields,
) ([]Check, error) {
	fileEntries, err := os.ReadDir(suitePath)
	if err != nil {
		return nil, fmt.Errorf("read suite dir %s: %w", suitePath, err)
	}

	var checks []Check
	for _, f := range fileEntries {
		if f.IsDir() || f.Name() == "metadata.yaml" {
			continue
		}
		fileName := f.Name()
		stem := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		filePath := filepath.Join(suitePath, fileName)

		fileMeta := readBestEffortCheckMeta(filePath)
		facets := resolveFacets(standardUID, suiteUID, suiteFacets, facetFields{
			Protocol: fileMeta.Protocol,
			SUT:      fileMeta.SUT,
			Role:     fileMeta.Role,
			Provider: fileMeta.Provider,
		})

		checks = append(checks, newSuiteCheck(
			standardUID, versionUID, suiteUID, stem, fileName,
			titleFromMeta(fileMeta, stem), visibleIn, facets,
		))
	}
	return checks, nil
}
