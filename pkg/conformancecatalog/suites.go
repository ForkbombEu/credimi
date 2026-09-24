// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"sort"
	"strings"
)

// SuiteMember is one check leaf on a suite-grain catalog row.
type SuiteMember struct {
	Path  string `json:"path"`
	Title string `json:"title"`
	File  string `json:"file"`
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

// projectSuites aggregates check rows into suite-grain records for the hub table.
// Suite display fields are looked up by PathPrefix from display (nil map = empty).
// providerLabels (from providers.yaml) fills ProviderLabel by provider slug.
// Suite row identity includes normalized standard/component/version plus suite uid,
// and keeps FS standard/version so the same suite under two profiles stays distinct.
func projectSuites(
	checks []Check,
	display map[string]suiteDisplayFields,
	providerLabels map[string]string,
) []SuiteRecord {
	type agg struct {
		meta     SuiteRecord
		vis      map[string]struct{}
		members  memberArray
		provider string
	}

	byKey := map[suiteAggKey]*agg{}
	var order []suiteAggKey

	for _, ch := range checks {
		key := suiteAggKey{
			standard:  ch.Standard,
			component: ch.Component,
			version:   ch.Version,
			suite:     ch.Suite,
			fsStd:     ch.FSStandard,
			fsVer:     ch.FSVersion,
		}
		a, ok := byKey[key]
		if !ok {
			prefix := suitePathPrefix(ch.FSStandard, ch.FSVersion, ch.Suite)
			disp := suiteDisplayFields{}
			if display != nil {
				disp = display[prefix]
			}
			providerLabel := ""
			if providerLabels != nil {
				providerLabel = providerLabels[ch.Provider]
			}
			a = &agg{
				meta: SuiteRecord{
					Standard:         ch.Standard,
					Component:        ch.Component,
					ComponentRank:    ComponentRank(ch.Component),
					Version:          ch.Version,
					Suite:            ch.Suite,
					Provider:         ch.Provider,
					ProviderLabel:    providerLabel,
					SuiteName:        disp.Name,
					SuiteSubtitle:    disp.Subtitle,
					SuiteHomepage:    disp.Homepage,
					SuiteRepository:  disp.Repository,
					SuiteHelp:        disp.Help,
					SuiteDescription: disp.Description,
					SuiteLogo:        disp.Logo,
					FSStandard:       ch.FSStandard,
					FSVersion:        ch.FSVersion,
					PathPrefix:       prefix,
				},
				vis:      map[string]struct{}{},
				provider: ch.Provider,
			}
			byKey[key] = a
			order = append(order, key)
		}
		a.members = append(a.members, SuiteMember{
			Path:  ch.Path,
			Title: ch.Title,
			File:  ch.File,
		})
		for _, v := range ch.VisibleIn {
			a.vis[v] = struct{}{}
		}
		if a.provider == "" && ch.Provider != "" {
			a.provider = ch.Provider
			a.meta.Provider = ch.Provider
			if providerLabels != nil {
				a.meta.ProviderLabel = providerLabels[ch.Provider]
			}
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
		rec.CheckCount = len(a.members)
		rec.Members = a.members
		rec.VisibleIn = stringArray(vis)
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
