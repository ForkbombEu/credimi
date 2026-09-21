// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import "strings"

// knownStandardFacets maps production standard directory UIDs to protocol/role
// when suite/check metadata omit explicit facet fields.
//
// This is a v1 bridge so PocketBase filters work without a full YAML backfill.
// Prefer in-file / suite metadata when present; do not grow this map as the
// long-term model (path layout redesign + authored facets come later).
var knownStandardFacets = map[string]struct {
	Protocol string
	Role     string
}{
	"openid4vp_wallet":   {Protocol: "openid4vp", Role: "wallet"},
	"openid4vp_verifier": {Protocol: "openid4vp", Role: "verifier"},
	"openid4vci_wallet":  {Protocol: "openid4vci", Role: "wallet"},
	"openid4vci_issuer":  {Protocol: "openid4vci", Role: "issuer"},
	"vlei":               {Protocol: "vlei"},
}

type facetFields struct {
	Protocol string
	SUT      string
	Role     string
	Provider string
}

// resolveFacets merges facet sources. Precedence (most specific wins):
// check/file meta → suite metadata.yaml → known standard UID map → suite UID as provider.
// FCAF checks typically supply sut/role in-file; provider defaults to "fcaf".
func resolveFacets(standardUID, suiteUID string, suiteMeta, fileMeta facetFields) facetFields {
	out := facetFields{
		Protocol: firstNonEmpty(fileMeta.Protocol, suiteMeta.Protocol),
		SUT:      firstNonEmpty(fileMeta.SUT, suiteMeta.SUT),
		Role:     firstNonEmpty(fileMeta.Role, suiteMeta.Role),
		Provider: firstNonEmpty(fileMeta.Provider, suiteMeta.Provider),
	}

	if known, ok := knownStandardFacets[standardUID]; ok {
		if out.Protocol == "" {
			out.Protocol = known.Protocol
		}
		if out.Role == "" {
			out.Role = known.Role
		}
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
