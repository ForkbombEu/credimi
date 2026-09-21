// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package conformancecatalog loads classic conformance checks and FCAF test
// definitions from config_templates into an in-process snapshot and projects
// them into a PocketBase collection used as an ephemeral query cache
// (filter/sort/pagination via native PB records API).
//
// Durable source of truth remains the filesystem under config_templates. The
// conformance_checks collection rows are always replaced on Rebuild and must not be
// treated as a second catalog of record (create/update/delete are rejected).
//
// Classic layout: standard/version/suite/<check file>.
// FCAF layout: fcaf/<version>/<suite>/tests/<id>.yaml — path identity is
// fcaf/<version>/<suite>/<test_id> (final segment = FCAF test id).
//
// Refresh after local template edits: restart the process (boot rebuild) or call
// Rebuild / POST /api/conformance-catalog/rebuild with the internal admin API key.
//
// Note: a shared :memory: ATTACH + SQLite VIEW projection is not viable — SQLite
// rejects views that reference attached databases. The PB collection table is
// therefore the query cache, with ephemeral lifecycle (full replace on rebuild).
package conformancecatalog

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"gopkg.in/yaml.v3"
)

const (
	// CollectionName is the PocketBase collection exposed to clients.
	CollectionName = "conformance_checks"

	// Stable collection id used by the migration and EnsureCollection.
	CollectionID = "pbc_conformance_checks_catalog"

	SurfaceManual   = "manual"
	SurfacePipeline = "pipeline"

	idNamespace = "credimi.conformance_check:"
)

// nonStandardTemplateDirs are skipped during the config_templates walk.
var nonStandardTemplateDirs = map[string]struct{}{
	"fcaf_sources": {},
}

// Check is one catalog entry derived from a template file.
type Check struct {
	ID               string   `json:"id"`
	Path             string   `json:"path"`
	Title            string   `json:"title"`
	Standard         string   `json:"standard"`
	Version          string   `json:"version"`
	Suite            string   `json:"suite"`
	File             string   `json:"file"`
	VisibleIn        []string `json:"visible_in"`
	Protocol         string   `json:"protocol"`
	SUT              string   `json:"sut"`
	Role             string   `json:"role"`
	Provider         string   `json:"provider"`
	StandardDisabled bool     `json:"-"`
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
	UID       string   `yaml:"uid"`
	VisibleIn []string `yaml:"visible_in"`
	Protocol  string   `yaml:"protocol"`
	SUT       string   `yaml:"sut"`
	Role      string   `yaml:"role"`
	Provider  string   `yaml:"provider"`
}

// Catalog holds the in-memory snapshot of checks loaded from disk.
type Catalog struct {
	mu      sync.RWMutex
	checks  []Check
	byID    map[string]Check
	byPath  map[string]Check
	rootDir string
}

var (
	defaultCatalog = &Catalog{}
	projecting     atomic.Bool
)

// Default returns the process-wide catalog instance.
func Default() *Catalog {
	return defaultCatalog
}

// Snapshot returns a copy of the current in-memory checks.
func (c *Catalog) Snapshot() []Check {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]Check, len(c.checks))
	copy(out, c.checks)
	return out
}

// LoadFromDir walks templatesDir, replaces the in-memory checks snapshot, and
// does not touch PocketBase. Useful for unit tests and handlers that need a
// snapshot without a full Rebuild transaction.
func (c *Catalog) LoadFromDir(templatesDir string) error {
	checks, err := loadFromDir(templatesDir)
	if err != nil {
		return err
	}
	c.replaceSnapshot(checks, templatesDir)
	return nil
}

// GetByID returns a check by stable record id.
func (c *Catalog) GetByID(id string) (Check, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ch, ok := c.byID[id]
	return ch, ok
}

// GetByPath returns a check by run path identity.
func (c *Catalog) GetByPath(path string) (Check, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ch, ok := c.byPath[path]
	return ch, ok
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

// LoadWalk walks templatesDir and returns flat conformance checks.
// Layout: standard/version/suite/file (classic) or …/tests/<id>.yaml (FCAF).
func LoadWalk(templatesDir string) ([]Check, error) {
	return loadFromDir(templatesDir)
}

// loadFromDir walks once and builds the flat check list with facet fields.
func loadFromDir(templatesDir string) ([]Check, error) {
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return nil, fmt.Errorf("read templates dir: %w", err)
	}

	var checks []Check

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
			if err := readYAML(filepath.Join(versionPath, "version.yaml"), &verMeta); err != nil {
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
				if err := readYAML(filepath.Join(suitePath, "metadata.yaml"), &sMeta); err != nil {
					return nil, err
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
					return nil, err
				}
				checks = append(checks, suiteChecks...)
			}
		}
	}

	return checks, nil
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
			Standard:         standardUID,
			Version:          versionUID,
			Suite:            suiteUID,
			File:             fileName,
			VisibleIn:        append([]string(nil), visibleIn...),
			Protocol:         facets.Protocol,
			SUT:              facets.SUT,
			Role:             facets.Role,
			Provider:         facets.Provider,
			StandardDisabled: standardDisabled,
		})
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
			Standard:         standardUID,
			Version:          versionUID,
			Suite:            suiteUID,
			File:             fileName,
			VisibleIn:        append([]string(nil), visibleIn...),
			Protocol:         facets.Protocol,
			SUT:              facets.SUT,
			Role:             facets.Role,
			Provider:         facets.Provider,
			StandardDisabled: standardDisabled,
		})
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

func (c *Catalog) replaceSnapshot(checks []Check, rootDir string) {
	byID := make(map[string]Check, len(checks))
	byPath := make(map[string]Check, len(checks))
	for _, ch := range checks {
		byID[ch.ID] = ch
		byPath[ch.Path] = ch
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks = checks
	c.byID = byID
	c.byPath = byPath
	c.rootDir = rootDir
}

// IsProjecting reports whether a Rebuild is currently writing collection rows.
func IsProjecting() bool {
	return projecting.Load()
}
