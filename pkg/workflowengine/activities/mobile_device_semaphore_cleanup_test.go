// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

type seededCleanupRecords struct {
	walletID, walletIdentifier         string
	credentialID, credentialIdentifier string
	useCaseID, useCaseIdentifier       string
}

func seedCleanupRecords(t *testing.T, app *tests.TestApp) seededCleanupRecords {
	t.Helper()
	wallet, walletIdentifier := tempRecordSeeders["wallet_versions"](t, app)
	credential, credentialIdentifier := tempRecordSeeders["credentials"](t, app)
	useCase, useCaseIdentifier := tempRecordSeeders["use_cases_verifications"](t, app)
	return seededCleanupRecords{
		walletID:             wallet.Id,
		walletIdentifier:     walletIdentifier,
		credentialID:         credential.Id,
		credentialIdentifier: credentialIdentifier,
		useCaseID:            useCase.Id,
		useCaseIdentifier:    useCaseIdentifier,
	}
}

func (s seededCleanupRecords) metadata(
	ownerID string,
) *mobiledevicesemaphore.MobileDeviceSemaphoreCleanupMetadata {
	return &mobiledevicesemaphore.MobileDeviceSemaphoreCleanupMetadata{
		TempWalletVersionID:         s.walletID,
		TempWalletVersionOwnerID:    ownerID,
		TempWalletVersionIdentifier: s.walletIdentifier,
		TempCredentials: []mobiledevicesemaphore.MobileDeviceSemaphoreTempCredentialCleanupMetadata{
			{RecordID: s.credentialID, OwnerID: testOrgAID, Identifier: s.credentialIdentifier},
		},
		TempUseCaseVerifications: []mobiledevicesemaphore.MobileDeviceSemaphoreTempCredentialCleanupMetadata{
			{RecordID: s.useCaseID, OwnerID: testOrgAID, Identifier: s.useCaseIdentifier},
		},
	}
}

func allSeededKept(s seededCleanupRecords) map[string]string {
	return map[string]string{
		"wallet_versions":         s.walletID,
		"credentials":             s.credentialID,
		"use_cases_verifications": s.useCaseID,
	}
}

func TestCleanupMobileDeviceSemaphoreResourcesActivity(t *testing.T) {
	cases := []struct {
		name         string
		cleanup      func(s seededCleanupRecords) *mobiledevicesemaphore.MobileDeviceSemaphoreCleanupMetadata
		wantFailures func(s seededCleanupRecords) []string
		wantKept     func(s seededCleanupRecords) map[string]string
	}{
		{
			name: "deletes every temporary record",
			cleanup: func(s seededCleanupRecords) *mobiledevicesemaphore.MobileDeviceSemaphoreCleanupMetadata {
				return s.metadata(testOrgAID)
			},
		},
		{
			name: "missing records are not failures",
			cleanup: func(seededCleanupRecords) *mobiledevicesemaphore.MobileDeviceSemaphoreCleanupMetadata {
				return &mobiledevicesemaphore.MobileDeviceSemaphoreCleanupMetadata{
					TempWalletVersionID:         "missingwallet12",
					TempWalletVersionOwnerID:    testOrgAID,
					TempWalletVersionIdentifier: testOrgANamespace + "/missing/wallet",
					TempCredentials: []mobiledevicesemaphore.MobileDeviceSemaphoreTempCredentialCleanupMetadata{
						{RecordID: "missingcred1234", OwnerID: testOrgAID, Identifier: "x"},
						{RecordID: "  "},
					},
					TempUseCaseVerifications: []mobiledevicesemaphore.MobileDeviceSemaphoreTempCredentialCleanupMetadata{
						{RecordID: "missingusecase1", OwnerID: testOrgAID, Identifier: "y"},
					},
				}
			},
			wantKept: allSeededKept,
		},
		{
			name: "owner mismatch is reported and the record is kept",
			cleanup: func(s seededCleanupRecords) *mobiledevicesemaphore.MobileDeviceSemaphoreCleanupMetadata {
				return s.metadata(testOrgBID)
			},
			wantFailures: func(s seededCleanupRecords) []string {
				return []string{
					"delete wallet_versions " + s.walletID +
						": temporary record mismatch: owner mismatch: wallet_versions " +
						s.walletID + " owner does not match expected_owner_id",
				}
			},
			wantKept: func(s seededCleanupRecords) map[string]string {
				return map[string]string{"wallet_versions": s.walletID}
			},
		},
		{
			name: "nil cleanup is a no-op",
			cleanup: func(seededCleanupRecords) *mobiledevicesemaphore.MobileDeviceSemaphoreCleanupMetadata {
				return nil
			},
			wantKept: allSeededKept,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newCredimiTestApp(t)
			seeded := seedCleanupRecords(t, app)

			result, err := executeActivity(
				t,
				NewCleanupMobileDeviceSemaphoreResourcesActivity(app),
				CleanupMobileDeviceSemaphoreResourcesActivityInput{Cleanup: tc.cleanup(seeded)},
			)
			require.NoError(t, err)

			output, err := workflowengine.DecodePayload[CleanupMobileDeviceSemaphoreResourcesActivityOutput](
				result.Output,
			)
			require.NoError(t, err)

			var wantFailures []string
			if tc.wantFailures != nil {
				wantFailures = tc.wantFailures(seeded)
			}
			require.Equal(t, wantFailures, output.CleanupFailures)

			kept := map[string]string{}
			if tc.wantKept != nil {
				kept = tc.wantKept(seeded)
			}
			for collection, recordID := range map[string]string{
				"wallet_versions":         seeded.walletID,
				"credentials":             seeded.credentialID,
				"use_cases_verifications": seeded.useCaseID,
			} {
				_, findErr := app.FindRecordById(collection, recordID)
				if kept[collection] != "" {
					require.NoError(t, findErr, collection)
				} else {
					require.Error(t, findErr, collection)
				}
			}
		})
	}
}

func TestCleanupMobileDeviceSemaphoreResourcesActivityInvalidPayload(t *testing.T) {
	app := newCredimiTestApp(t)
	_, err := executeActivity(
		t,
		NewCleanupMobileDeviceSemaphoreResourcesActivity(app),
		"not-an-object",
	)
	require.Error(t, err)
}
