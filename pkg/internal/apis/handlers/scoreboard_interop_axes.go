// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"sort"
	"strings"
)

type interopAxis struct {
	HubCollection string
	CacheField    string
	PathBased     bool
}

var interopAxisRegistry = map[string]interopAxis{
	"wallets": {
		HubCollection: "wallets",
		CacheField:    "wallets",
		PathBased:     false,
	},
	"credential_issuers": {
		HubCollection: "credential_issuers",
		CacheField:    "issuers",
		PathBased:     false,
	},
	"credentials": {
		HubCollection: "credentials",
		CacheField:    "credentials",
		PathBased:     false,
	},
	"verifiers": {
		HubCollection: "verifiers",
		CacheField:    "verifiers",
		PathBased:     false,
	},
	"use_cases_verifications": {
		HubCollection: "use_cases_verifications",
		CacheField:    "use_case_verifications",
		PathBased:     false,
	},
	"conformance-checks": {
		HubCollection: "conformance-checks",
		CacheField:    "conformance_checks",
		PathBased:     true,
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
