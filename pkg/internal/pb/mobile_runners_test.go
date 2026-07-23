// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	tclient "go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func TestRegisterMobileRunnerHooksDeletesRunnerByShuttingDownSemaphore(t *testing.T) {
	origClient := mobileRunnerShutdownTemporalClient
	t.Cleanup(func() { mobileRunnerShutdownTemporalClient = origClient })

	mockClient := temporalmocks.NewClient(t)
	mockClient.On(
		"UpdateWorkflow",
		mock.Anything,
		mock.MatchedBy(func(options tclient.UpdateWorkflowOptions) bool {
			req, ok := options.Args[0].(workflows.MobileDeviceSemaphoreShutdownDeviceRequest)
			return options.WorkflowID == workflows.MobileDeviceSemaphoreWorkflowID(
				"usera-s-organization/runner-delete/device-a",
			) &&
				options.UpdateName == workflows.MobileDeviceSemaphoreShutdownDeviceUpdate &&
				options.UpdateID == "shutdown/usera-s-organization/runner-delete/device-a" &&
				options.WaitForStage == tclient.WorkflowUpdateStageAccepted &&
				ok &&
				req.Reason == "mobile runner deleted"
		}),
	).Return(temporalmocks.NewWorkflowUpdateHandle(t), nil).Once()
	mobileRunnerShutdownTemporalClient = func(namespace string) (tclient.Client, error) {
		require.Equal(t, workflowengine.MobileDeviceSemaphoreDefaultNamespace, namespace)
		return mockClient, nil
	}

	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	ensureMobileDevicesCollection(t, app)
	canonify.RegisterCanonifyHooks(app)
	RegisterMobileRunnerHooks(app)

	record := createMobileRunnerRecordForDeleteTest(t, app, "runner-delete")
	createMobileDeviceRecordForDeleteTest(t, app, record, "device-a")
	require.NoError(t, app.Delete(record))
}

func TestRegisterMobileRunnerHooksIgnoresMissingSemaphore(t *testing.T) {
	origClient := mobileRunnerShutdownTemporalClient
	t.Cleanup(func() { mobileRunnerShutdownTemporalClient = origClient })

	mockClient := temporalmocks.NewClient(t)
	mockClient.On("UpdateWorkflow", mock.Anything, mock.Anything).
		Return(nil, &serviceerror.NotFound{Message: "missing"}).Once()
	mobileRunnerShutdownTemporalClient = func(namespace string) (tclient.Client, error) {
		return mockClient, nil
	}

	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	ensureMobileDevicesCollection(t, app)
	canonify.RegisterCanonifyHooks(app)
	RegisterMobileRunnerHooks(app)

	record := createMobileRunnerRecordForDeleteTest(t, app, "runner-missing")
	createMobileDeviceRecordForDeleteTest(t, app, record, "device-a")
	require.NoError(t, app.Delete(record))
}

func ensureMobileDevicesCollection(t *testing.T, app core.App) {
	t.Helper()
	if _, err := app.FindCollectionByNameOrId("mobile_devices"); err == nil {
		return
	}

	collection := core.NewBaseCollection("mobile_devices")
	collection.Fields.Add(
		&core.RelationField{
			Name:         "owner",
			CollectionId: "aako88kt3br4npt",
			MaxSelect:    1,
			Required:     true,
		},
	)
	collection.Fields.Add(
		&core.RelationField{
			Name:          "runner",
			CollectionId:  "pbc_500646217",
			MaxSelect:     1,
			Required:      true,
			CascadeDelete: true,
		},
	)
	collection.Fields.Add(&core.TextField{Name: "name", Required: true})
	collection.Fields.Add(&core.TextField{Name: "canonified_name", Required: true})
	require.NoError(t, app.Save(collection))
}

func createMobileDeviceRecordForDeleteTest(
	t *testing.T,
	app core.App,
	runner *core.Record,
	name string,
) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("mobile_devices")
	require.NoError(t, err)
	record := core.NewRecord(collection)
	record.Set("owner", runner.GetString("owner"))
	record.Set("runner", runner.Id)
	record.Set("name", name)
	record.Set("canonified_name", name)
	require.NoError(t, app.Save(record))
}

func createMobileRunnerRecordForDeleteTest(t *testing.T, app core.App, name string) *core.Record {
	t.Helper()

	orgID, err := getOrgIDfromName(app)
	require.NoError(t, err)
	orgRecord, err := app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	if orgRecord.GetString("canonified_name") == "" {
		orgRecord.Set("canonified_name", "usera-s-organization")
		require.NoError(t, app.Save(orgRecord))
	}

	runnersColl, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	record := core.NewRecord(runnersColl)
	record.Set("owner", orgID)
	record.Set("name", name)
	record.Set("canonified_name", name)
	record.Set("ip", "127.0.0.1")
	record.Set("type", "android_emulator")
	record.Set("runner_url", "https://runner.test")
	record.Set("serial", "serial-1")
	require.NoError(t, app.Save(record))
	return record
}
