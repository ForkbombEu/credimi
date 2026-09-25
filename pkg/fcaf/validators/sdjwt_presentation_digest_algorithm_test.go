// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func digestSession(status string, algorithms ...string) map[string]any {
	session := map[string]any{"status": status}
	if len(algorithms) == 0 {
		return session
	}
	entries := make([]any, 0, len(algorithms))
	for _, algorithm := range algorithms {
		entries = append(entries, map[string]any{
			"format":           "dc+sd-jwt",
			"digest_algorithm": algorithm,
		})
	}
	session["decoded_presentations"] = map[string]any{"pid_sha384": entries}
	return session
}

// TestSDJWTPresentationDigestAlgorithmValidator separates the two outcomes the
// source distinguishes: a Wallet that withholds the credential satisfies the
// requirement, while a Wallet that presents it has shown it supports the hash
// function and therefore falls outside the source's profile applicability.
func TestSDJWTPresentationDigestAlgorithmValidator(t *testing.T) {
	cases := []struct {
		name    string
		session any
		params  map[string]any
		want    Status
	}{
		{
			name:    "wallet withheld the credential",
			session: digestSession("request_retrieved"),
			want:    StatusPass,
		},
		{
			name:    "wallet presented a sha-256 credential instead",
			session: digestSession("presentation_validated", "sha-256"),
			want:    StatusPass,
		},
		{
			name:    "wallet presented the sha-384 credential",
			session: digestSession("presentation_validated", "sha-384"),
			want:    StatusNotApplicable,
		},
		{
			name:    "wallet presented several credentials, one of them sha-384",
			session: digestSession("presentation_validated", "sha-256", "sha-384"),
			want:    StatusNotApplicable,
		},
		{
			name:    "session evidence carries no status",
			session: map[string]any{"decoded_presentations": map[string]any{}},
			want:    StatusFail,
		},
		{
			name:    "session evidence is missing",
			session: nil,
			want:    StatusFail,
		},
		{
			name:    "digest_algorithm param is absent",
			session: digestSession("request_retrieved"),
			params:  map[string]any{},
			want:    StatusError,
		},
	}

	validator := SDJWTPresentationDigestAlgorithmValidator{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params := tc.params
			if params == nil {
				params = map[string]any{"digest_algorithm": "sha-384"}
			}
			result := validator.Validate(
				context.Background(),
				Input{Value: tc.session, Params: params},
			)
			require.Equal(t, tc.want, result.Status, result.Message)
		})
	}
}
