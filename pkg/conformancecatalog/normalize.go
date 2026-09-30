// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

// PathIdentity is the in-memory product projection of a filesystem catalog path.
// Durable path strings stay FS-shaped; hub grouping/filter/sort uses these fields.
type PathIdentity struct {
	// Standard is the product standard uid (openid4vp, openid4vci, or passthrough).
	Standard string
	// Component is wallet, issuer, or verifier when known; empty when unmapped.
	Component string
	// Version is the standard profile version (1.0, draft-23, …). Empty for FCAF
	// packs that have no pinned OpenID profile yet (FS version is an SUT label).
	Version string
}

// classicFSPrefix maps config_templates top-level dirs that encode standard×component.
var classicFSPrefix = map[string]struct {
	standard  string
	component string
}{
	"openid4vp_wallet":   {standard: "openid4vp", component: "wallet"},
	"openid4vp_verifier": {standard: "openid4vp", component: "verifier"},
	"openid4vci_wallet":  {standard: "openid4vci", component: "wallet"},
	"openid4vci_issuer":  {standard: "openid4vci", component: "issuer"},
}

// fcafPackIdentity maps known FCAF (version, suite) packs to product axes.
// EC party strings (e.g. relying_party) are not Credimi components.
var fcafPackIdentity = map[string]map[string]PathIdentity{
	"wallet_solution": {
		"relying_party": {
			Standard:  "openid4vp",
			Component: "wallet",
			Version:   "", // no pinned OpenID4VP profile for this Commission pack yet
		},
	},
}

// NormalizePathIdentity projects FS path segments into product Standard / Component /
// Version without renaming durable paths.
func NormalizePathIdentity(fsStandard, fsVersion, fsSuite string) PathIdentity {
	if mapped, ok := classicFSPrefix[fsStandard]; ok {
		return PathIdentity{
			Standard:  mapped.standard,
			Component: mapped.component,
			Version:   fsVersion,
		}
	}

	if fsStandard == "fcaf" {
		if bySuite, ok := fcafPackIdentity[fsVersion]; ok {
			if id, ok := bySuite[fsSuite]; ok {
				return id
			}
		}
	}

	// Unmapped tops (vlei, unknown FCAF packs): keep FS labels; no invented component.
	return PathIdentity{
		Standard:  fsStandard,
		Component: "",
		Version:   fsVersion,
	}
}
