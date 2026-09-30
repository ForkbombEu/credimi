// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"fmt"
	"os"
	"path/filepath"
)

// suiteWork is one discovered Conformance suite root on the Filesystem axis
// (standard/version/suite with authored metadata.yaml). Check loading
// (classic vs FCAF) stays in the layout adapters.
type suiteWork struct {
	suitePath   string
	standardUID string
	versionUID  string
	suiteUID    string
	visibleIn   []string
	suiteFacets facetFields
	display     suiteDisplayFields
}

// enumerateSuites walks templatesDir and returns every suite root that passes
// Filesystem-axis discovery rules: skip nonStandardTemplateDirs; soft-missing
// standard.yaml OK; version.yaml and metadata.yaml must exist (Stat gate);
// authored YAML unmarshal fails closed.
func enumerateSuites(templatesDir string) ([]suiteWork, error) {
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return nil, fmt.Errorf("read templates dir: %w", err)
	}

	var suites []suiteWork
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, skip := nonStandardTemplateDirs[entry.Name()]; skip {
			continue
		}

		standardUID := entry.Name()
		standardPath := filepath.Join(templatesDir, standardUID)

		stdMeta := standardYAML{UID: standardUID}
		if err := readRequiredYAML(filepath.Join(standardPath, "standard.yaml"), &stdMeta); err != nil {
			return nil, err
		}
		if stdMeta.UID == "" {
			stdMeta.UID = standardUID
		}

		versionEntries, err := os.ReadDir(standardPath)
		if err != nil {
			return nil, fmt.Errorf("read standard dir %s: %w", standardPath, err)
		}

		for _, vEntry := range versionEntries {
			if !vEntry.IsDir() {
				continue
			}
			versionUID := vEntry.Name()
			versionPath := filepath.Join(standardPath, versionUID)
			if _, err := os.Stat(filepath.Join(versionPath, "version.yaml")); err != nil {
				continue
			}

			verMeta := versionYAML{UID: versionUID}
			if err := readRequiredYAML(filepath.Join(versionPath, "version.yaml"), &verMeta); err != nil {
				return nil, err
			}
			if verMeta.UID == "" {
				verMeta.UID = versionUID
			}

			suiteEntries, err := os.ReadDir(versionPath)
			if err != nil {
				return nil, fmt.Errorf("read version dir %s: %w", versionPath, err)
			}

			for _, sEntry := range suiteEntries {
				if !sEntry.IsDir() {
					continue
				}
				suiteUID := sEntry.Name()
				suitePath := filepath.Join(versionPath, suiteUID)
				if _, err := os.Stat(filepath.Join(suitePath, "metadata.yaml")); err != nil {
					continue
				}

				sMeta := suiteYAML{UID: suiteUID}
				if err := readRequiredYAML(filepath.Join(suitePath, "metadata.yaml"), &sMeta); err != nil {
					return nil, err
				}
				if sMeta.UID == "" {
					sMeta.UID = suiteUID
				}

				suites = append(suites, suiteWork{
					suitePath:   suitePath,
					standardUID: stdMeta.UID,
					versionUID:  verMeta.UID,
					suiteUID:    sMeta.UID,
					visibleIn:   normalizeVisibleIn(sMeta.VisibleIn),
					suiteFacets: facetFields{
						Protocol: sMeta.Protocol,
						SUT:      sMeta.SUT,
						Role:     sMeta.Role,
						Provider: sMeta.Provider,
					},
					display: suiteDisplayFromYAML(sMeta),
				})
			}
		}
	}
	return suites, nil
}
