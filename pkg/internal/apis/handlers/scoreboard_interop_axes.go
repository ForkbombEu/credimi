// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"sort"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

type interopAxisTier struct {
	GroupCollection string
	LeafCollection  string
	LeafCacheField  string
	LeafParentField string
	NoLeafSentinel  string
}

type interopAxis struct {
	HubCollection string
	CacheField    string
	PathBased     bool
	Tier          *interopAxisTier
	buildEntity   func(app core.App, axisRecord *core.Record, cacheRecord *core.Record) (InteropMatrixEntity, error)
}

func (a interopAxis) Tiered() bool {
	return a.Tier != nil
}

var interopAxisRegistry = map[string]interopAxis{
	"wallets": {
		HubCollection: "wallets",
		CacheField:    "wallets",
		PathBased:     false,
		buildEntity:   walletBuildEntity,
		Tier: &interopAxisTier{
			GroupCollection: "wallets",
			LeafCollection:  "wallet_versions",
			LeafCacheField:  "wallet_versions",
			LeafParentField: "wallet",
			NoLeafSentinel:  "__no_version__",
		},
	},
	"credential_issuers": {
		HubCollection: "credential_issuers",
		CacheField:    "issuers",
		PathBased:     false,
		buildEntity:   credentialIssuerBuildEntity,
		Tier: &interopAxisTier{
			GroupCollection: "credential_issuers",
			LeafCollection:  "credentials",
			LeafCacheField:  "credentials",
			LeafParentField: "credential_issuer",
			NoLeafSentinel:  "__no_leaf__",
		},
	},
	"credentials": {
		HubCollection: "credentials",
		CacheField:    "credentials",
		PathBased:     false,
		buildEntity:   credentialBuildEntity,
	},
	"verifiers": {
		HubCollection: "verifiers",
		CacheField:    "verifiers",
		PathBased:     false,
		buildEntity:   verifierBuildEntity,
		Tier: &interopAxisTier{
			GroupCollection: "verifiers",
			LeafCollection:  "use_case_verifications",
			LeafCacheField:  "use_case_verifications",
			LeafParentField: "verifier",
			NoLeafSentinel:  "__no_leaf__",
		},
	},
	"use_cases_verifications": {
		HubCollection: "use_cases_verifications",
		CacheField:    "use_case_verifications",
		PathBased:     false,
		buildEntity:   useCaseVerificationBuildEntity,
	},
	"conformance-checks": {
		HubCollection: "conformance-checks",
		CacheField:    "conformance_checks",
		PathBased:     true,
		Tier: &interopAxisTier{
			GroupCollection: "conformance-checks",
			LeafCollection:  "conformance-checks",
			LeafCacheField:  "conformance_checks",
			LeafParentField: "",
			NoLeafSentinel:  "",
		},
	},
}

func getInteropAxis(hubCollection string) (interopAxis, bool) {
	axis, ok := interopAxisRegistry[hubCollection]
	return axis, ok
}

func supportedInteropHubCollections() []string {
	hubs := make([]string, 0, len(interopAxisRegistry))
	for hub := range interopAxisRegistry {
		hubs = append(hubs, hub)
	}
	sort.Strings(hubs)
	return hubs
}

func interopHubsUsageHint() string {
	return "use row= and column= with hub collections: " +
		strings.Join(supportedInteropHubCollections(), ", ")
}
