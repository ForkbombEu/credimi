// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import "testing"

func TestNormalizeProtocolAndAuthor(t *testing.T) {
	tests := []struct {
		name           string
		protocol       string
		author         string
		expectedProto  string
		expectedAuthor string
	}{
		{
			name:           "Normalize openid4vp_wallet protocol",
			protocol:       "openid4vp_wallet",
			author:         "some_author",
			expectedProto:  "OpenID4VP_Wallet",
			expectedAuthor: "some_author",
		},
		{
			name:           "Normalize openid4vci_wallet protocol",
			protocol:       "openid4vci_wallet",
			author:         "some_author",
			expectedProto:  "OpenID4VCI_Wallet",
			expectedAuthor: "some_author",
		},
		{
			name:           "Normalize openid_foundation author",
			protocol:       "some_protocol",
			author:         "openid_foundation",
			expectedProto:  "some_protocol",
			expectedAuthor: "OpenID_foundation",
		},
		{
			name:           "No normalization needed",
			protocol:       "some_protocol",
			author:         "some_author",
			expectedProto:  "some_protocol",
			expectedAuthor: "some_author",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProto, gotAuthor := normalizeProtocolAndAuthor(tt.protocol, tt.author)
			if gotProto != tt.expectedProto {
				t.Errorf(
					"normalizeProtocolAndAuthor() protocol = %v, want %v",
					gotProto,
					tt.expectedProto,
				)
			}
			if gotAuthor != tt.expectedAuthor {
				t.Errorf(
					"normalizeProtocolAndAuthor() author = %v, want %v",
					gotAuthor,
					tt.expectedAuthor,
				)
			}
		})
	}
}
