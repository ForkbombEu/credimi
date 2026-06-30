// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func TestRegisterMobileRunnerWorkerManagerHooks_RunnerPublishDispatchesToPublishedOrgs(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	ensureWorkerManagerPublicationFields(t, app)
	canonify.RegisterCanonifyHooks(app)

	orgID, err := getOrgIDfromName(app)
	require.NoError(t, err)
	orgA, err := app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	orgA.Set("published", true)
	require.NoError(t, app.Save(orgA))

	orgsColl, err := app.FindCollectionByNameOrId("organizations")
	require.NoError(t, err)
	orgB := core.NewRecord(orgsColl)
	orgB.Set("name", "Org B")
	orgB.Set("canonified_name", "org-b")
	orgB.Set("published", true)
	require.NoError(t, app.Save(orgB))

	runner := createWorkerManagerRunnerRecord(
		t,
		app,
		orgID,
		"public-later",
		"https://runner.example",
		false,
		false,
	)

	origStartManager := startWorkerManagerFn
	t.Cleanup(func() {
		startWorkerManagerFn = origStartManager
	})

	calls := make(chan string, 2)
	startWorkerManagerFn = func(_ core.App, namespace, oldNamespace string, runnerURLs []string) {
		require.Empty(t, oldNamespace)
		require.Equal(t, []string{"https://runner.example"}, runnerURLs)
		calls <- namespace
	}
	RegisterMobileRunnerWorkerManagerHooks(app)

	runner.Set("published", true)
	require.NoError(t, app.Save(runner))

	got := []string{<-calls, <-calls}
	require.ElementsMatch(t, []string{orgA.GetString("canonified_name"), "org-b"}, got)
}
