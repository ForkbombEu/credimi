// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStepDefinition_UnmarshalJSON_FlatWith(t *testing.T) {
	input := `{
		"id": "step1",
		"use": "mobile_automation",
		"with": {
			"action_id": "onboarding-0001",
			"version_id": "v1"
		}
	}`

	var step StepDefinition
	err := json.Unmarshal([]byte(input), &step)
	require.NoError(t, err)

	require.NotEmpty(t, step.With.Payload)
	require.Equal(t, "onboarding-0001", step.With.Payload["action_id"])
}
