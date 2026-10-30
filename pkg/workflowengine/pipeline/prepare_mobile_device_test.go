// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"errors"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

const (
	prepareTestDeviceID = "tenant/runner-1/device-1"
	prepareTestVersion  = "tenant/wallet/v1"
)

type prepareMobileDeviceTestResult struct {
	Prepared PreparedMobileDevice `json:"prepared"`
	Error    string               `json:"error"`
	Tracked  map[string]any       `json:"tracked"`
}

// registerPrepareMobileDeviceWorkflow runs PrepareMobileDevice and returns
// both its outcome and the tracked devices, so a failure can be inspected.
func registerPrepareMobileDeviceWorkflow(env *testsuite.TestWorkflowEnvironment) {
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (prepareMobileDeviceTestResult, error) {
			settedDevices := map[string]any{}
			prepared, err := PrepareMobileDevice(
				ctx,
				workflow.ActivityOptions{StartToCloseTimeout: time.Second},
				settedDevices,
				MobileDeviceRequest{
					DeviceID:          prepareTestDeviceID,
					VersionIdentifier: prepareTestVersion,
					StepID:            "action-author",
					RunIdentifier:     "session-1",
				},
			)
			result := prepareMobileDeviceTestResult{Prepared: prepared, Tracked: settedDevices}
			if err != nil {
				result.Error = err.Error()
			}
			return result, nil
		},
		workflow.RegisterOptions{Name: "test-prepare-mobile-device"},
	)
}

func registerPrepareMobileDeviceActivities(env *testsuite.TestWorkflowEnvironment) {
	registerStubActivity(env, activities.GetMobileDeviceActivityName)
	registerMobileRunnerHTTPActivity(env)
	for _, name := range []string{
		iosPlatformActivities.Start,
		iosPlatformActivities.Install,
		iosPlatformActivities.PostInstall,
		androidPlatformActivities.Start,
		androidPlatformActivities.Install,
		androidPlatformActivities.PostInstall,
		activities.NewListInstalledAppsActivity().Name(),
	} {
		registerStubActivity(env, name)
	}
}

func onPrepareInstallerAction(env *testsuite.TestWorkflowEnvironment, installerPath string) {
	env.OnActivity(
		activities.NewMobileRunnerHTTPActivity(nil).Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}
			body, ok := payload["body"].(map[string]any)
			return ok &&
				workflowengine.AsString(payload["path"]) == "/credimi/installer-action" &&
				workflowengine.AsString(body["version_identifier"]) == prepareTestVersion
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"installer_path": installerPath,
			"version_id":     prepareTestVersion,
		},
	}}, nil)
}

func onPrepareIOSSimulator(env *testsuite.TestWorkflowEnvironment) {
	onGetMobileDevice(env, prepareTestDeviceID, map[string]any{
		"runner_id":  "tenant/runner-1",
		"runner_url": "https://runner.example",
		"type":       "ios_simulator",
	})
	env.OnActivity(iosPlatformActivities.Start, mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"name":   "clone-1",
			"serial": "UDID-CLONE-1",
		}}, nil)
	env.OnActivity(activities.NewListInstalledAppsActivity().Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: []string{"com.apple.Preferences"}}, nil)
}

func runPrepareMobileDevice(
	t *testing.T,
	env *testsuite.TestWorkflowEnvironment,
) prepareMobileDeviceTestResult {
	t.Helper()
	env.ExecuteWorkflow("test-prepare-mobile-device")
	require.NoError(t, env.GetWorkflowError())
	var result prepareMobileDeviceTestResult
	require.NoError(t, env.GetWorkflowResult(&result))
	return result
}

func TestPrepareMobileDeviceReturnsPreparedIOSSimulator(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	registerPrepareMobileDeviceActivities(env)
	registerPrepareMobileDeviceWorkflow(env)
	onPrepareIOSSimulator(env)
	onPrepareInstallerAction(env, "/runner/wallet.app")
	installed := workflowengine.ActivityResult{Output: map[string]any{"package_id": "eu.wallet"}}
	env.OnActivity(
		iosPlatformActivities.Install,
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			return ok && payload["app"] == "/runner/wallet.app" &&
				payload["serial"] == "UDID-CLONE-1"
		}),
	).Return(installed, nil)
	env.OnActivity(iosPlatformActivities.PostInstall, mock.Anything, mock.Anything).
		Return(installed, nil)

	result := runPrepareMobileDevice(t, env)

	require.Empty(t, result.Error)
	require.Equal(t, PreparedMobileDevice{
		RunnerID:      "tenant/runner-1",
		Serial:        "UDID-CLONE-1",
		Type:          "ios_simulator",
		PackageID:     "eu.wallet",
		InstallerPath: "/runner/wallet.app",
	}, result.Prepared)
	require.Contains(t, result.Tracked, prepareTestDeviceID)
}

func TestPrepareMobileDeviceTracksDeviceBeforeRunnerActivities(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	registerPrepareMobileDeviceActivities(env)
	registerPrepareMobileDeviceWorkflow(env)
	onGetMobileDevice(env, prepareTestDeviceID, map[string]any{
		"runner_id":  "tenant/runner-1",
		"runner_url": "https://runner.example",
		"type":       "ios_simulator",
	})
	env.OnActivity(iosPlatformActivities.Start, mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, errors.New("simulator unavailable"))

	result := runPrepareMobileDevice(t, env)

	require.Contains(t, result.Error, "simulator unavailable")
	require.Contains(t, result.Tracked, prepareTestDeviceID)
}

func TestPrepareMobileDeviceRequiresIOSInstallerPath(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	registerPrepareMobileDeviceActivities(env)
	registerPrepareMobileDeviceWorkflow(env)
	onPrepareIOSSimulator(env)
	onPrepareInstallerAction(env, "")
	installed := workflowengine.ActivityResult{Output: map[string]any{"package_id": "eu.wallet"}}
	env.OnActivity(iosPlatformActivities.Install, mock.Anything, mock.Anything).
		Return(installed, nil)
	env.OnActivity(iosPlatformActivities.PostInstall, mock.Anything, mock.Anything).
		Return(installed, nil)

	result := runPrepareMobileDevice(t, env)

	require.Contains(t, result.Error, "missing installer path for "+prepareTestVersion)
}

func TestPrepareMobileDeviceRequiresInstalledPackage(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	registerPrepareMobileDeviceActivities(env)
	registerPrepareMobileDeviceWorkflow(env)
	onGetMobileDevice(env, prepareTestDeviceID, map[string]any{
		"runner_id":  "tenant/runner-1",
		"runner_url": "https://runner.example",
		"type":       "android_emulator",
	})
	env.OnActivity(androidPlatformActivities.Start, mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"name":   "avd-1",
			"serial": "emulator-5554",
		}}, nil)
	env.OnActivity(activities.NewListInstalledAppsActivity().Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: []string{}}, nil)
	// The runner answers for another version, so nothing is installed for the requested one.
	env.OnActivity(activities.NewMobileRunnerHTTPActivity(nil).Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"body": map[string]any{
				"installer_path": "/runner/wallet.apk",
				"version_id":     "tenant/wallet/other",
			},
		}}, nil)
	installed := workflowengine.ActivityResult{Output: map[string]any{"package_id": "eu.wallet"}}
	env.OnActivity(androidPlatformActivities.Install, mock.Anything, mock.Anything).
		Return(installed, nil)
	env.OnActivity(androidPlatformActivities.PostInstall, mock.Anything, mock.Anything).
		Return(installed, nil)

	result := runPrepareMobileDevice(t, env)

	require.Contains(t, result.Error, "missing installed package for "+prepareTestVersion)
}
