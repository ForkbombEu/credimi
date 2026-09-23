// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"fmt"
	"sort"
	"strings"
)

// SuiteRecord is one hub-table row: a suite under normalized standard×component×version.
// FS* fields preserve durable path identity for hub links; Standard/Component/Version
// are the product projection.
type SuiteRecord struct {
	ID               string   `json:"id"`
	Standard         string   `json:"standard"`  // normalized product standard
	Component        string   `json:"component"` // wallet | issuer | verifier | ""
	ComponentRank    int      `json:"component_rank"` // wallet=0, issuer=1, verifier=2, else=9
	Version          string   `json:"version"`        // normalized profile version (may be "")
	Suite            string   `json:"suite"`          // suite uid
	Provider         string   `json:"provider"`
	SuiteName        string   `json:"suite_name"`
	SuiteHomepage    string   `json:"suite_homepage"`
	SuiteRepository  string   `json:"suite_repository"`
	SuiteHelp        string   `json:"suite_help"`
	SuiteDescription string   `json:"suite_description"`
	SuiteLogo        string   `json:"suite_logo"`
	CheckCount       int      `json:"check_count"`
	CheckPaths       []string `json:"check_paths"`
	CheckTitles      []string `json:"check_titles"`
	CheckFiles       []string `json:"check_files"`
	VisibleIn        []string `json:"visible_in"`
	FSStandard       string   `json:"fs_standard"`
	FSVersion        string   `json:"fs_version"`
	PathPrefix       string   `json:"path_prefix"` // fs_standard/fs_version/suite for hub URLs
}

// ComponentRank returns the hub default sort priority for a component uid.
// wallet → issuer → verifier; unknown/empty last.
func ComponentRank(component string) int {
	switch strings.TrimSpace(component) {
	case "wallet":
		return 0
	case "issuer":
		return 1
	case "verifier":
		return 2
	default:
		return 9
	}
}

type suiteAggKey struct {
	standard  string
	component string
	version   string
	suite     string
	fsStd     string
	fsVer     string
}

// ProjectSuites aggregates check rows into suite-grain records for the hub table.
// Suite row identity includes normalized standard/component/version plus suite uid,
// and keeps FS standard/version so the same suite under two profiles stays distinct.
func ProjectSuites(checks []Check) []SuiteRecord {
	type agg struct {
		meta     SuiteRecord
		vis      map[string]struct{}
		paths    []string
		titles   []string
		files    []string
		provider string
	}

	byKey := map[suiteAggKey]*agg{}
	var order []suiteAggKey

	for _, ch := range checks {
		key := suiteAggKey{
			standard:  ch.NormStandard,
			component: ch.Component,
			version:   ch.NormVersion,
			suite:     ch.Suite,
			fsStd:     ch.Standard,
			fsVer:     ch.Version,
		}
		a, ok := byKey[key]
		if !ok {
			a = &agg{
				meta: SuiteRecord{
					Standard:         ch.NormStandard,
					Component:        ch.Component,
					ComponentRank:    ComponentRank(ch.Component),
					Version:          ch.NormVersion,
					Suite:            ch.Suite,
					Provider:         ch.Provider,
					SuiteName:        ch.SuiteName,
					SuiteHomepage:    ch.SuiteHomepage,
					SuiteRepository:  ch.SuiteRepository,
					SuiteHelp:        ch.SuiteHelp,
					SuiteDescription: ch.SuiteDescription,
					SuiteLogo:        ch.SuiteLogo,
					FSStandard:       ch.Standard,
					FSVersion:        ch.Version,
					PathPrefix:       fmt.Sprintf("%s/%s/%s", ch.Standard, ch.Version, ch.Suite),
				},
				vis:      map[string]struct{}{},
				provider: ch.Provider,
			}
			byKey[key] = a
			order = append(order, key)
		}
		a.paths = append(a.paths, ch.Path)
		a.titles = append(a.titles, ch.Title)
		a.files = append(a.files, ch.File)
		for _, v := range ch.VisibleIn {
			a.vis[v] = struct{}{}
		}
		if a.provider == "" && ch.Provider != "" {
			a.provider = ch.Provider
			a.meta.Provider = ch.Provider
		}
		if a.meta.SuiteName == "" && ch.SuiteName != "" {
			a.meta.SuiteName = ch.SuiteName
		}
		if a.meta.SuiteLogo == "" && ch.SuiteLogo != "" {
			a.meta.SuiteLogo = ch.SuiteLogo
		}
	}

	out := make([]SuiteRecord, 0, len(order))
	for _, key := range order {
		a := byKey[key]
		vis := make([]string, 0, len(a.vis))
		for v := range a.vis {
			vis = append(vis, v)
		}
		sort.Strings(vis)

		rec := a.meta
		rec.ID = PathID(rec.PathPrefix)
		rec.Provider = a.provider
		rec.CheckCount = len(a.paths)
		rec.CheckPaths = a.paths
		rec.CheckTitles = a.titles
		rec.CheckFiles = a.files
		rec.VisibleIn = vis
		out = append(out, rec)
	}

	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.ComponentRank != b.ComponentRank {
			return a.ComponentRank < b.ComponentRank
		}
		if c := strings.Compare(a.Standard, b.Standard); c != 0 {
			return c < 0
		}
		if c := strings.Compare(a.Suite, b.Suite); c != 0 {
			return c < 0
		}
		if c := strings.Compare(a.Version, b.Version); c != 0 {
			return c < 0
		}
		return a.PathPrefix < b.PathPrefix
	})

	return out
}
