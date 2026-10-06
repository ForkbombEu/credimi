// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCredimiActivitiesRegistersEveryNameOnce(t *testing.T) {
	names := make([]string, 0, 10)
	for _, act := range CredimiActivities(nil) {
		names = append(names, act.Name())
	}
	require.ElementsMatch(t, []string{
		ResolveRecordActivityName,
		GetCredentialOfferActivityName,
		GetUseCaseVerificationDeeplinkActivityName,
		DeleteTempRecordActivityName,
		GetMobileDeviceActivityName,
		ValidateDeviceAccessActivityName,
		SendRealtimeLogsActivityName,
		StoreCredentialIssuerActivityName,
		StoreIssuerCredentialActivityName,
		DeletePipelineResultFilesActivityName,
	}, names)
}
