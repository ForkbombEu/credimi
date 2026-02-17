// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/stretchr/testify/require"
)

func TestDecodeAndValidatePayload(t *testing.T) {
	_, err := decodeAndValidatePayload(&StepDefinition{StepSpec: StepSpec{ID: "step-1", With: StepInputs{Payload: map[string]any{}}}})
	require.Error(t, err)

	_, err = decodeAndValidatePayload(&StepDefinition{StepSpec: StepSpec{ID: "step-2", With: StepInputs{Payload: map[string]any{
		"action_code": "code",
	}}}})
	require.Error(t, err)

	_, err = decodeAndValidatePayload(&StepDefinition{StepSpec: StepSpec{ID: "step-3", With: StepInputs{Payload: map[string]any{
		"runner_id": "runner-1",
	}}}})
	require.Error(t, err)

	payload, err := decodeAndValidatePayload(&StepDefinition{StepSpec: StepSpec{ID: "step-4", With: StepInputs{Payload: map[string]any{
		"action_id": "action-1",
		"runner_id": "runner-2",
	}}}})
	require.NoError(t, err)
	require.Equal(t, "action-1", payload.ActionID)
	require.Equal(t, "runner-2", payload.RunnerID)
}

func TestCollectMobileRunnerIDs(t *testing.T) {
	steps := []StepDefinition{
		{StepSpec: StepSpec{Use: mobileAutomationStepUse, With: StepInputs{Payload: map[string]any{
			"action_id": "action-1",
			"runner_id": "runner-b",
		}}}},
		{StepSpec: StepSpec{Use: mobileAutomationStepUse, With: StepInputs{Payload: map[string]any{
			"action_id": "action-2",
			"runner_id": "runner-a",
		}}}},
	}

	runnerIDs, err := collectMobileRunnerIDs(steps, "runner-global")
	require.NoError(t, err)
	require.Equal(t, []string{"runner-a", "runner-b", "runner-global"}, runnerIDs)
}

func TestHasRunnerPermit(t *testing.T) {
	runData := map[string]any{
		"mobile_runner_permits": map[string]workflows.MobileRunnerSemaphorePermit{
			"runner-1": {RunnerID: "runner-1"},
		},
	}
	permit := hasRunnerPermit(&runData, "runner-1")
	require.True(t, permit)
	require.False(t, hasRunnerPermit(&runData, "runner-2"))
}

func TestGetRunnerPermits(t *testing.T) {
	runData := map[string]any{
		"mobile_runner_permits": map[string]any{
			"runner-1": map[string]any{"runner_id": "runner-1", "lease_id": "lease-1"},
			"bad":      "nope",
		},
	}
	permits := getRunnerPermits(runData)
	require.Len(t, permits, 1)
	require.Equal(t, "runner-1", permits["runner-1"].RunnerID)
}

func TestParseAPKResponse(t *testing.T) {
	result := workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"apk_path": "path.apk",
			"version_id": "ver-1",
			"code": "action-code",
		},
	}}
	payload := &workflows.MobileAutomationWorkflowPipelinePayload{ActionID: "action-1"}
	step := &StepDefinition{StepSpec: StepSpec{ID: "step-1"}}
	apkPath, versionID, actionCode, err := parseAPKResponse(result, payload, step)
	require.NoError(t, err)
	require.Equal(t, "path.apk", apkPath)
	require.Equal(t, "ver-1", versionID)
	require.Equal(t, "action-code", actionCode)

	badResult := workflowengine.ActivityResult{Output: map[string]any{"body": map[string]any{}}}
	_, _, _, err = parseAPKResponse(badResult, payload, step)
	require.Error(t, err)
}

func TestGetOrCreateSettedDevices(t *testing.T) {
	runData := map[string]any{}
	devices := getOrCreateSettedDevices(&runData)
	require.NotNil(t, devices)
	require.Empty(t, devices)

	runData["setted_devices"] = map[string]any{"runner": map[string]any{"serial": "1"}}
	restored := getOrCreateSettedDevices(&runData)
	require.Contains(t, restored, "runner")
}

func TestIsSemaphoreManagedRun(t *testing.T) {
	require.False(t, isSemaphoreManagedRun(nil))
	require.False(t, isSemaphoreManagedRun(map[string]any{}))
	require.True(t, isSemaphoreManagedRun(map[string]any{mobileRunnerSemaphoreTicketIDConfigKey: "ticket"}))
}
