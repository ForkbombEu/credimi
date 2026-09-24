// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizePathIdentity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		fsStandard string
		fsVersion  string
		fsSuite    string
		want       PathIdentity
	}{
		{
			name:       "openid4vp wallet classic",
			fsStandard: "openid4vp_wallet",
			fsVersion:  "draft-24",
			fsSuite:    "openid_conformance_suite",
			want: PathIdentity{
				Standard:  "openid4vp",
				Component: "wallet",
				Version:   "draft-24",
			},
		},
		{
			name:       "openid4vp verifier",
			fsStandard: "openid4vp_verifier",
			fsVersion:  "1.0",
			fsSuite:    "webuild",
			want: PathIdentity{
				Standard:  "openid4vp",
				Component: "verifier",
				Version:   "1.0",
			},
		},
		{
			name:       "openid4vci wallet",
			fsStandard: "openid4vci_wallet",
			fsVersion:  "1.0",
			fsSuite:    "webuild",
			want: PathIdentity{
				Standard:  "openid4vci",
				Component: "wallet",
				Version:   "1.0",
			},
		},
		{
			name:       "openid4vci issuer",
			fsStandard: "openid4vci_issuer",
			fsVersion:  "1.0",
			fsSuite:    "webuild",
			want: PathIdentity{
				Standard:  "openid4vci",
				Component: "issuer",
				Version:   "1.0",
			},
		},
		{
			name:       "fcaf relying party pack maps to openid4vp wallet; version empty",
			fsStandard: "fcaf",
			fsVersion:  "wallet_solution",
			fsSuite:    "relying_party",
			want: PathIdentity{
				Standard:  "openid4vp",
				Component: "wallet",
				Version:   "",
			},
		},
		{
			name:       "same suite uid under two versions stays distinct by version",
			fsStandard: "openid4vp_wallet",
			fsVersion:  "1.0",
			fsSuite:    "openid_conformance_suite",
			want: PathIdentity{
				Standard:  "openid4vp",
				Component: "wallet",
				Version:   "1.0",
			},
		},
		{
			name:       "unmapped vlei passthrough standard; empty component",
			fsStandard: "vlei",
			fsVersion:  "1.0",
			fsSuite:    "acme",
			want: PathIdentity{
				Standard:  "vlei",
				Component: "",
				Version:   "1.0",
			},
		},
		{
			name:       "unknown fcaf pack does not invent component",
			fsStandard: "fcaf",
			fsVersion:  "other_sut",
			fsSuite:    "other_party",
			want: PathIdentity{
				Standard:  "fcaf",
				Component: "",
				Version:   "other_sut",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := NormalizePathIdentity(tc.fsStandard, tc.fsVersion, tc.fsSuite)
			require.Equal(t, tc.want, got)
		})
	}
}
