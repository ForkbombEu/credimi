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

	"gopkg.in/yaml.v3"
)

const (
	// CollectionName is the fake PocketBase collection name used in URLs and
	// client code (pb.collection('conformance_checks')).
	CollectionName = "conformance_checks"

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

// Check is one catalog entry derived from a template file.
// Lean check grain: path, title, facets, and identity only. Suite display
// metadata (name, logo, URLs) lives on the suite projection (ADR-0002).
type Check struct {
	ID         string   `json:"id"`
	Path       string   `json:"path"`
	Title      string   `json:"title"`
	FSStandard string   `json:"fs_standard"` // FS top-level dir (durable path segment)
	FSVersion  string   `json:"fs_version"`  // FS version segment (durable)
	Suite      string   `json:"suite"`
	File       string   `json:"file"`
	VisibleIn  []string `json:"visible_in"`
	Protocol   string   `json:"protocol"`
	SUT        string   `json:"sut"`
	Role       string   `json:"role"`
	Provider   string   `json:"provider"`
	// Product projection — same wire names as SuiteRecord (ADR-0002 dual axes).
	Standard         string `json:"standard"`
	Component        string `json:"component"`
	Version          string `json:"version"`
	StandardDisabled bool   `json:"-"`
}

// LoadedCatalog is the filesystem walk result: lean checks plus suite display
// keyed for suite projection (ADR-0002).
type LoadedCatalog struct {
	Checks       []Check
	SuiteDisplay map[string]suiteDisplayFields // key = fs_standard/fs_version/suite
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

// walk-only YAML shapes (not exposed as nested API DTOs).
type standardYAML struct {
	UID      string `yaml:"uid"`
	Disabled bool   `yaml:"disabled"`
}

type versionYAML struct {
	UID string `yaml:"uid"`
}

type suiteYAML struct {
	UID         string   `yaml:"uid"`
	Name        string   `yaml:"name"`
	Homepage    string   `yaml:"homepage"`
	Repository  string   `yaml:"repository"`
	Help        string   `yaml:"help"`
	Description string   `yaml:"description"`
	Logo        string   `yaml:"logo"`
	VisibleIn   []string `yaml:"visible_in"`
	Protocol    string   `yaml:"protocol"`
	SUT         string   `yaml:"sut"`
	Role        string   `yaml:"role"`
	Provider    string   `yaml:"provider"`
}

// suiteDisplayFields are authored suite metadata keyed for suite projection
// (ADR-0002), not denormalized onto checks.
type suiteDisplayFields struct {
	Name        string
	Homepage    string
	Repository  string
	Help        string
	Description string
	Logo        string
}

func suiteDisplayFromYAML(s suiteYAML) suiteDisplayFields {
	return suiteDisplayFields{
		Name:        strings.TrimSpace(s.Name),
		Homepage:    strings.TrimSpace(s.Homepage),
		Repository:  strings.TrimSpace(s.Repository),
		Help:        strings.TrimSpace(s.Help),
		Description: strings.TrimSpace(s.Description),
		Logo:        strings.TrimSpace(s.Logo),
	}
}

// Catalog is a thin facade over the ephemeral query cache. It holds no
// separate check slice — Rebuild writes only to :memory: SQLite, and Snapshot
// reads that projection (for tests that still assert on []Check).
type Catalog struct{}

var defaultCatalog = &Catalog{}

// Default returns the process-wide catalog facade.
func Default() *Catalog {
	return defaultCatalog
}

// Snapshot returns checks from the ephemeral :memory: cache (SELECT → []Check).
// On cache errors it returns nil so callers observe an empty catalog rather
// than a stale dual-written slice.
func (c *Catalog) Snapshot() []Check {
	_ = c
	checks, err := listEphemeralChecks()
	if err != nil {
		return nil
	}
	return checks
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

type checkFileMeta struct {
	Title    string `yaml:"title"`
	Name     string `yaml:"name"`
	Protocol string `yaml:"protocol"`
	SUT      string `yaml:"sut"`
	Role     string `yaml:"role"`
	Provider string `yaml:"provider"`
}

// LoadFromDir walks templatesDir once and returns lean checks plus suite display
// metadata. Layout: standard/version/suite/file (classic) or …/tests/<id>.yaml
// (FCAF). Does not touch PocketBase; Rebuild calls this then projects into the
// process-private ephemeral store.
func LoadFromDir(templatesDir string) (LoadedCatalog, error) {
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return LoadedCatalog{}, fmt.Errorf("read templates dir: %w", err)
	}

	loaded := LoadedCatalog{
		SuiteDisplay: map[string]suiteDisplayFields{},
	}

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
		if err := readYAML(filepath.Join(standardPath, "standard.yaml"), &stdMeta); err != nil {
			return LoadedCatalog{}, err
		}
		if stdMeta.UID == "" {
			stdMeta.UID = standardUID
		}

		versionEntries, err := os.ReadDir(standardPath)
		if err != nil {
			return LoadedCatalog{}, fmt.Errorf("read standard dir %s: %w", standardPath, err)
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
			if err := readYAML(filepath.Join(versionPath, "version.yaml"), &verMeta); err != nil {
				return LoadedCatalog{}, err
			}
			if verMeta.UID == "" {
				verMeta.UID = versionUID
			}

			suiteEntries, err := os.ReadDir(versionPath)
			if err != nil {
				return LoadedCatalog{}, fmt.Errorf("read version dir %s: %w", versionPath, err)
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
				if err := readYAML(filepath.Join(suitePath, "metadata.yaml"), &sMeta); err != nil {
					return LoadedCatalog{}, err
				}
				if sMeta.UID == "" {
					sMeta.UID = suiteUID
				}

				visibleIn := normalizeVisibleIn(sMeta.VisibleIn)
				suiteFacets := facetFields{
					Protocol: sMeta.Protocol,
					SUT:      sMeta.SUT,
					Role:     sMeta.Role,
					Provider: sMeta.Provider,
				}
				loaded.SuiteDisplay[suitePathPrefix(stdMeta.UID, verMeta.UID, sMeta.UID)] =
					suiteDisplayFromYAML(sMeta)

				var suiteChecks []Check
				if hasFCAFTestsDir(stdMeta.UID, suitePath) {
					suiteChecks, err = loadFCAFSuiteTests(
						suitePath,
						stdMeta.UID,
						verMeta.UID,
						sMeta.UID,
						visibleIn,
						stdMeta.Disabled,
						suiteFacets,
					)
				} else {
					suiteChecks, err = loadClassicSuiteChecks(
						suitePath,
						stdMeta.UID,
						verMeta.UID,
						sMeta.UID,
						visibleIn,
						stdMeta.Disabled,
						suiteFacets,
					)
				}
				if err != nil {
					return LoadedCatalog{}, err
				}
				loaded.Checks = append(loaded.Checks, suiteChecks...)
			}
		}
	}

	return loaded, nil
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

func loadClassicSuiteChecks(
	suitePath, standardUID, versionUID, suiteUID string,
	visibleIn []string,
	standardDisabled bool,
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
		path := fmt.Sprintf("%s/%s/%s/%s", standardUID, versionUID, suiteUID, stem)
		filePath := filepath.Join(suitePath, fileName)

		fileMeta := checkFileMeta{}
		_ = readYAML(filePath, &fileMeta)
		facets := resolveFacets(standardUID, suiteUID, suiteFacets, facetFields{
			Protocol: fileMeta.Protocol,
			SUT:      fileMeta.SUT,
			Role:     fileMeta.Role,
			Provider: fileMeta.Provider,
		})

		checks = append(checks, Check{
			ID:               PathID(path),
			Path:             path,
			Title:            titleFromMeta(fileMeta, stem),
			FSStandard:       standardUID,
			FSVersion:        versionUID,
			Suite:            suiteUID,
			File:             fileName,
			VisibleIn:        append([]string(nil), visibleIn...),
			Protocol:         facets.Protocol,
			SUT:              facets.SUT,
			Role:             facets.Role,
			Provider:         facets.Provider,
			StandardDisabled: standardDisabled,
		}.withNormalizedIdentity())
	}
	return checks, nil
}

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

// loadFCAFSuiteTests indexes FCAF definitions from suite/tests/*.yaml.
// Path identity is fcaf/<version>/<suite>/<test_id> so nest/pickers stay stable;
// the final path segment is the FCAF test id used by fcaf-validation.test_ids.
func loadFCAFSuiteTests(
	suitePath, standardUID, versionUID, suiteUID string,
	visibleIn []string,
	standardDisabled bool,
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
		if err := readYAML(filePath, &meta); err != nil {
			return nil, err
		}
		testID := strings.TrimSpace(meta.ID)
		if testID == "" {
			testID = strings.TrimSuffix(fileName, filepath.Ext(fileName))
		}
		if testID == "" {
			continue
		}

		path := fmt.Sprintf("%s/%s/%s/%s", standardUID, versionUID, suiteUID, testID)
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

		checks = append(checks, Check{
			ID:               PathID(path),
			Path:             path,
			Title:            title,
			FSStandard:       standardUID,
			FSVersion:        versionUID,
			Suite:            suiteUID,
			File:             fileName,
			VisibleIn:        append([]string(nil), visibleIn...),
			Protocol:         facets.Protocol,
			SUT:              facets.SUT,
			Role:             facets.Role,
			Provider:         facets.Provider,
			StandardDisabled: standardDisabled,
		}.withNormalizedIdentity())
	}
	return checks, nil
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

func titleFromMeta(meta checkFileMeta, stem string) string {
	if title := strings.TrimSpace(meta.Title); title != "" {
		return title
	}
	if name := strings.TrimSpace(meta.Name); name != "" {
		return name
	}
	return filenameTitle(stem)
}

func filenameTitle(stem string) string {
	if stem == "" {
		return "untitled"
	}
	return stem
}

func readYAML(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		// Check files may not be YAML metadata-first; ignore parse errors for title extraction.
		if _, ok := out.(*checkFileMeta); ok {
			return nil
		}
		return fmt.Errorf("unmarshal %s: %w", path, err)
	}
	return nil
}
