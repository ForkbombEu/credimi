// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package conformancecatalog loads classic conformance checks and FCAF test
// definitions from config_templates into a process-private :memory: SQLite
// query cache (the sole post-rebuild projection).
//
// Durable source of truth remains the filesystem under config_templates.
// There is no durable catalog collection in PocketBase data.db.
//
// Clients use PocketBase collection URL shapes:
//   - /api/collections/conformance_checks/records (check grain)
//   - /api/collections/conformance_suites/records (suite grain; display metadata)
//
// Credimi owns those routes and runs filter/sort/pagination via
// pocketbase/tools/search against the ephemeral DB. Writes are rejected.
// List/get auth is public (same posture as the former blueprints/hub listing).
//
// Classic layout: standard/version/suite/<check file>.
// FCAF layout: fcaf/<version>/<suite>/tests/<id>.yaml — path identity is
// fcaf/<version>/<suite>/<test_id> (final segment = FCAF test id).
//
// Refresh after local template edits: restart the process (boot rebuild) or call
// Rebuild / POST /api/conformance-catalog/rebuild with the internal admin API key.
package conformancecatalog

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// ChecksCollectionName is the fake PocketBase collection name used in URLs and
	// client code (pb.collection('conformance_checks')).
	ChecksCollectionName = "conformance_checks"

	// CollectionID is a stable id echoed in JSON for PocketBase client compatibility.
	CollectionID = "pbc_conformance_checks_catalog"

	// SuitesCollectionName is the fake PocketBase collection for suite-grain rows.
	SuitesCollectionName = "conformance_suites"

	// SuitesCollectionID is echoed in suite list/get JSON for PB client compatibility.
	SuitesCollectionID = "pbc_conformance_suites_catalog"

	SurfaceManual   = "manual"
	SurfacePipeline = "pipeline"

	idNamespace = "credimi.conformance_check:"
)

// nonStandardTemplateDirs are skipped during the config_templates walk.
var nonStandardTemplateDirs = map[string]struct{}{
	"fcaf_sources": {},
}

// LoadedCatalog is the filesystem walk result: lean checks and projected suite
// rows (ADR-0002, ADR-0006). Suite display metadata stays internal to the walk.
// Check / SuiteRecord types are generated from columnSpec (ADR-0009).
type LoadedCatalog struct {
	Checks []Check
	Suites []SuiteRecord
}

func suitePathPrefix(fsStd, fsVer, suite string) string {
	return fmt.Sprintf("%s/%s/%s", fsStd, fsVer, suite)
}

// withNormalizedIdentity fills Standard / Component / Version from FS segments.
func (ch Check) withNormalizedIdentity() Check {
	id := NormalizePathIdentity(ch.FSStandard, ch.FSVersion, ch.Suite)
	ch.Standard = id.Standard
	ch.Component = id.Component
	ch.Version = id.Version
	return ch
}

// TemplatesDir resolves config_templates from ROOT_DIR (empty ROOT_DIR → ./config_templates).
func TemplatesDir() string {
	root := os.Getenv("ROOT_DIR")
	if root == "" {
		return "config_templates"
	}
	return filepath.Join(root, "config_templates")
}

// PathID returns the stable PocketBase record id for a check path.
// Encoding: lowercase hex SHA-256 of "credimi.conformance_check:" + path, truncated to 15
// characters to match PocketBase's default id alphabet/length ([a-z0-9]{15}).
func PathID(path string) string {
	sum := sha256.Sum256([]byte(idNamespace + path))
	return hex.EncodeToString(sum[:])[:15]
}

// LoadFromDir walks templatesDir once and returns lean checks plus projected
// suite-grain rows. Layout: standard/version/suite/file (classic) or
// …/tests/<id>.yaml (FCAF). Suite display from metadata.yaml is applied during
// projection and is not part of the returned value. Does not touch PocketBase;
// Rebuild inserts the returned rows into the process-private ephemeral store.
func LoadFromDir(templatesDir string) (LoadedCatalog, error) {
	suites, err := enumerateSuites(templatesDir)
	if err != nil {
		return LoadedCatalog{}, err
	}

	providerLabels, err := loadProviderLabels(templatesDir)
	if err != nil {
		return LoadedCatalog{}, err
	}

	var checks []Check
	suiteDisplay := map[string]suiteDisplayFields{}

	for _, s := range suites {
		suiteDisplay[suitePathPrefix(s.standardUID, s.versionUID, s.suiteUID)] = s.display

		var suiteChecks []Check
		if hasFCAFTestsDir(s.standardUID, s.suitePath) {
			suiteChecks, err = loadFCAFSuiteTests(
				s.suitePath,
				s.standardUID,
				s.versionUID,
				s.suiteUID,
				s.visibleIn,
				s.suiteFacets,
			)
		} else {
			suiteChecks, err = loadClassicSuiteChecks(
				s.suitePath,
				s.standardUID,
				s.versionUID,
				s.suiteUID,
				s.visibleIn,
				s.suiteFacets,
			)
		}
		if err != nil {
			return LoadedCatalog{}, err
		}
		checks = append(checks, suiteChecks...)
	}

	return LoadedCatalog{
		Checks: checks,
		Suites: projectSuites(checks, suiteDisplay, providerLabels),
	}, nil
}

// newSuiteCheck builds one lean check row after a layout loader has resolved
// path stem, file name, title, and facets (classic vs FCAF stay separate).
func newSuiteCheck(
	fsStandard, fsVersion, suite, stem, fileName, title string,
	visibleIn []string,
	facets facetFields,
) Check {
	path := fmt.Sprintf("%s/%s/%s/%s", fsStandard, fsVersion, suite, stem)
	return Check{
		ID:         PathID(path),
		Path:       path,
		Title:      title,
		FSStandard: fsStandard,
		FSVersion:  fsVersion,
		Suite:      suite,
		File:       fileName,
		VisibleIn:  append(stringArray(nil), visibleIn...),
		Protocol:   facets.Protocol,
		SUT:        facets.SUT,
		Role:       facets.Role,
		Provider:   facets.Provider,
	}.withNormalizedIdentity()
}

func normalizeVisibleIn(visibleIn []string) []string {
	if len(visibleIn) == 0 {
		return []string{SurfaceManual, SurfacePipeline}
	}
	out := make([]string, 0, len(visibleIn))
	seen := map[string]struct{}{}
	for _, v := range visibleIn {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	if len(out) == 0 {
		return []string{SurfaceManual, SurfacePipeline}
	}
	return out
}
