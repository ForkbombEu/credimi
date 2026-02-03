// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const testDataDir = "../../../test_pb_data"

func setupSchedulesTestApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	RegisterSchedulesHooks(app)
	return app
}

func TestSchedulesEnrichMissingOwner(t *testing.T) {
	app := setupSchedulesTestApp(t)
	defer app.Cleanup()

	collection, err := app.FindCollectionByNameOrId("schedules")
	require.NoError(t, err)

	record := core.NewRecord(collection)
	record.Set("owner", "")

	event := &core.RecordEnrichEvent{App: app}
	event.Record = record
	err = app.OnRecordEnrich("schedules").Trigger(event)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to fetch owner organization")
}

func TestSchedulesEnrichInvalidOwner(t *testing.T) {
	app := setupSchedulesTestApp(t)
	defer app.Cleanup()

	collection, err := app.FindCollectionByNameOrId("schedules")
	require.NoError(t, err)

	record := core.NewRecord(collection)
	record.Set("owner", "missing-owner")

	event := &core.RecordEnrichEvent{App: app}
	event.Record = record
	err = app.OnRecordEnrich("schedules").Trigger(event)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to fetch owner organization")
}
