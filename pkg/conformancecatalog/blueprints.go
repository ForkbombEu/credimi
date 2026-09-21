// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

// Nested blueprints DTO matches GET /api/template/blueprints consumers.
// Built once during LoadWalk/Rebuild; request path only projects + filters.

// StandardMetadata is standard.yaml for a blueprints standard node.
type StandardMetadata struct {
	UID           string              `json:"uid"            yaml:"uid"`
	Name          string              `json:"name"           yaml:"name"`
	Description   string              `json:"description"    yaml:"description"`
	StandardURL   string              `json:"standard_url"   yaml:"standard_url"`
	LatestUpdate  string              `json:"latest_update"  yaml:"latest_update"`
	ExternalLinks map[string][]string `json:"external_links" yaml:"external_links"`
	Disabled      bool                `json:"disabled"       yaml:"disabled"`
}

// VersionMetadata is version.yaml for a blueprints version node.
type VersionMetadata struct {
	UID              string `json:"uid"               yaml:"uid"`
	Name             string `json:"name"              yaml:"name"`
	LatestUpdate     string `json:"latest_update"     yaml:"latest_update"`
	SpecificationURL string `json:"specification_url" yaml:"specification_url"`
}

// SuiteMetadata is metadata.yaml for a blueprints suite node.
// VisibleIn is the raw suite value (empty means both surfaces); omitempty preserves
// prior JSON shape for default-visibility suites.
type SuiteMetadata struct {
	UID         string   `json:"uid"                  yaml:"uid"`
	Name        string   `json:"name"                 yaml:"name"`
	Homepage    string   `json:"homepage"             yaml:"homepage"`
	Repository  string   `json:"repository"           yaml:"repository"`
	Help        string   `json:"help"                 yaml:"help"`
	Description string   `json:"description"          yaml:"description"`
	VisibleIn   []string `json:"visible_in,omitempty" yaml:"visible_in,omitempty"`
	Logo        string   `json:"logo"                 yaml:"logo"`
}

// Suite is a blueprints suite with check file names and path identities.
type Suite struct {
	SuiteMetadata
	Files []string `json:"files" yaml:"files"`
	Paths []string `json:"paths" yaml:"paths"`
}

// Version is a blueprints version with suites.
type Version struct {
	VersionMetadata
	Suites []Suite `json:"suites" yaml:"suites"`
}

// Standard is a blueprints standard with versions.
type Standard struct {
	StandardMetadata
	Versions []Version `json:"versions" yaml:"versions"`
}

// Blueprints is the nested /api/template/blueprints response body.
type Blueprints []Standard

// ProjectBlueprints returns a surface-filtered copy of the nested tree.
// Empty surface returns the full tree (still a deep copy of suite slices).
func ProjectBlueprints(tree Blueprints, surface string) Blueprints {
	if surface == "" {
		return cloneBlueprints(tree)
	}

	out := make(Blueprints, 0, len(tree))
	for _, std := range tree {
		versions := make([]Version, 0, len(std.Versions))
		for _, ver := range std.Versions {
			suites := make([]Suite, 0, len(ver.Suites))
			for _, suite := range ver.Suites {
				if !suiteVisibleIn(suite.SuiteMetadata, surface) {
					continue
				}
				suites = append(suites, cloneSuite(suite))
			}
			if len(suites) == 0 {
				continue
			}
			versions = append(versions, Version{
				VersionMetadata: ver.VersionMetadata,
				Suites:          suites,
			})
		}
		if len(versions) == 0 {
			continue
		}
		out = append(out, Standard{
			StandardMetadata: std.StandardMetadata,
			Versions:         versions,
		})
	}
	return out
}

func suiteVisibleIn(suiteMeta SuiteMetadata, surface string) bool {
	if len(suiteMeta.VisibleIn) == 0 {
		return surface == SurfaceManual || surface == SurfacePipeline
	}
	for _, visibleSurface := range suiteMeta.VisibleIn {
		if visibleSurface == surface {
			return true
		}
	}
	return false
}

func cloneBlueprints(tree Blueprints) Blueprints {
	out := make(Blueprints, 0, len(tree))
	for _, std := range tree {
		versions := make([]Version, len(std.Versions))
		for i, ver := range std.Versions {
			suites := make([]Suite, len(ver.Suites))
			for j, suite := range ver.Suites {
				suites[j] = cloneSuite(suite)
			}
			versions[i] = Version{
				VersionMetadata: ver.VersionMetadata,
				Suites:          suites,
			}
		}
		out = append(out, Standard{
			StandardMetadata: std.StandardMetadata,
			Versions:         versions,
		})
	}
	return out
}

func cloneSuite(suite Suite) Suite {
	files := []string{}
	if len(suite.Files) > 0 {
		files = append(files, suite.Files...)
	}
	paths := []string{}
	if len(suite.Paths) > 0 {
		paths = append(paths, suite.Paths...)
	}
	var visible []string
	if suite.VisibleIn != nil {
		visible = append([]string{}, suite.VisibleIn...)
	}
	meta := suite.SuiteMetadata
	meta.VisibleIn = visible
	return Suite{
		SuiteMetadata: meta,
		Files:         files,
		Paths:         paths,
	}
}
