// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"errors"
	"net/http"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

func TestRegisterPipelineHooksPreventsPublishedPipelineChanges(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	RegisterPipelineHooks(app)

	pipelineRecord := createTestPipelineRecord(t, app)

	pipelineRecord.Set("description", "changed while private")
	require.NoError(t, app.Save(pipelineRecord))

	pipelineRecord.Set("published", true)
	require.NoError(t, app.Save(pipelineRecord))

	pipelineRecord.Set("description", "changed while published")
	err = app.Save(pipelineRecord)
	require.Error(t, err)

	var apiErr *router.ApiError
	require.True(t, errors.As(err, &apiErr))
	require.Equal(t, http.StatusBadRequest, apiErr.Status)
	require.Contains(t, apiErr.Message, "Published pipeline records cannot be changed")

	reloaded, err := app.FindRecordById(pipelinesCollectionName, pipelineRecord.Id)
	require.NoError(t, err)
	require.Equal(t, "changed while private", reloaded.GetString("description"))
}

func TestRegisterPipelineHooksAllowsUnpublishing(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	RegisterPipelineHooks(app)
	canonify.RegisterCanonifyHooks(app)

	pipelineRecord := createTestPipelineRecord(t, app)
	pipelineRecord.Set("published", true)
	require.NoError(t, app.Save(pipelineRecord))

	pipelineRecord.Set("published", false)
	pipelineRecord.Set("canonified_name", "")
	require.NoError(t, app.Save(pipelineRecord))

	pipelineRecord.Set("description", "changed after unpublish")
	require.NoError(t, app.Save(pipelineRecord))
}

func createTestPipelineRecord(t *testing.T, app core.App) *core.Record {
	t.Helper()

	orgID, err := getOrgIDfromName(app)
	require.NoError(t, err)

	pipelinesColl, err := app.FindCollectionByNameOrId(pipelinesCollectionName)
	require.NoError(t, err)

	pipelineRecord := core.NewRecord(pipelinesColl)
	pipelineRecord.Set("owner", orgID)
	pipelineRecord.Set("name", "hook-test-pipeline")
	pipelineRecord.Set("canonified_name", "hook-test-pipeline")
	pipelineRecord.Set("description", "test pipeline")
	pipelineRecord.Set("yaml", "name: hook-test-pipeline\nsteps: []\n")
	require.NoError(t, app.Save(pipelineRecord))

	return pipelineRecord
}
