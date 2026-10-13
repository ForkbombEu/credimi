// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunner

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
)

func TestCredential(t *testing.T) {
	coll := core.NewBaseCollection("mobile_runners")
	newRunner := func(id string, generation int) *core.Record {
		record := core.NewRecord(coll)
		record.Id = id
		record.Set("credential_generation", generation)
		return record
	}

	for _, tc := range []struct {
		name    string
		secret  string
		runner  *core.Record
		want    string
		wantErr error
	}{
		{
			// Pinned vector: the runner side only compares values, but every
			// Credimi instance sharing a secret must derive the same one.
			name:   "hmac of record id and generation",
			secret: "test-secret",
			runner: newRunner("abc123", 2),
			want:   "7030b40ba2573ff4d4d4e3b0da0b860da15311dd870c97f2c478a39c554ae7e1",
		},
		{
			name:   "next generation rotates the credential",
			secret: "test-secret",
			runner: newRunner("abc123", 3),
			want:   "4362a16645a4b6d9b738b91519afc1f09f6c27e2d40638706fbdfb025a1933f3",
		},
		{
			name:    "missing secret",
			secret:  " ",
			runner:  newRunner("abc123", 2),
			wantErr: ErrCredentialSecretMissing,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(CredentialSecretEnvVar, tc.secret)
			got, err := Credential(tc.runner)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Empty(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestCredentialDiffersPerRunner(t *testing.T) {
	t.Setenv(CredentialSecretEnvVar, "test-secret")
	coll := core.NewBaseCollection("mobile_runners")
	first := core.NewRecord(coll)
	first.Id = "runner-a"
	second := core.NewRecord(coll)
	second.Id = "runner-b"

	firstCredential, err := Credential(first)
	require.NoError(t, err)
	secondCredential, err := Credential(second)
	require.NoError(t, err)
	require.NotEqual(t, firstCredential, secondCredential)
}
