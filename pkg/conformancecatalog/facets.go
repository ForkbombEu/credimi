// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import "strings"

type facetFields struct {
	Protocol string
	SUT      string
	Role     string
	Provider string
}

// resolveFacets merges facet sources. Precedence (most specific wins):
// check/file meta → suite metadata.yaml → suite UID as provider (FCAF → "fcaf").
// Classic suites author protocol/role/provider in metadata.yaml; sut stays empty.
// FCAF checks typically supply sut/role in-file; provider defaults to "fcaf".
func resolveFacets(standardUID, suiteUID string, suiteMeta, fileMeta facetFields) facetFields {
	out := facetFields{
		Protocol: firstNonEmpty(fileMeta.Protocol, suiteMeta.Protocol),
		SUT:      firstNonEmpty(fileMeta.SUT, suiteMeta.SUT),
		Role:     firstNonEmpty(fileMeta.Role, suiteMeta.Role),
		Provider: firstNonEmpty(fileMeta.Provider, suiteMeta.Provider),
	}

	if out.Provider == "" {
		if standardUID == "fcaf" {
			out.Provider = "fcaf"
		} else if suiteUID != "" {
			out.Provider = suiteUID
		}
	}

	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
