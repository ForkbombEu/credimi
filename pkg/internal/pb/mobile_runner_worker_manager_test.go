// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func TestRegisterMobileRunnerWorkerManagerHooks_RunnerPublishDispatchesToPublishedOrgs(
	t *testing.T,
) {
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

func TestRegisterMobileRunnerWorkerManagerHooks_RunnerEligibilityGating(t *testing.T) {
	testCases := []struct {
		name        string
		prepare     func(record *core.Record)
		update      func(record *core.Record)
		wantStarted bool
	}{
		{
			name:        "disabled runner publication does not start workers",
			prepare:     func(record *core.Record) { record.Set("disabled", true) },
			update:      func(record *core.Record) { record.Set("published", true) },
			wantStarted: false,
		},
		{
			name:        "offline runner publication does not start workers",
			prepare:     func(record *core.Record) { record.Set("online", false) },
			update:      func(record *core.Record) { record.Set("published", true) },
			wantStarted: false,
		},
		{
			name: "re-enabling a published runner starts workers",
			prepare: func(record *core.Record) {
				record.Set("published", true)
				record.Set("disabled", true)
			},
			update:      func(record *core.Record) { record.Set("disabled", false) },
			wantStarted: true,
		},
		{
			name: "runner returning online starts workers",
			prepare: func(record *core.Record) {
				record.Set("published", true)
				record.Set("online", false)
			},
			update:      func(record *core.Record) { record.Set("online", true) },
			wantStarted: true,
		},
		{
			name:        "already startable runner is not restarted",
			prepare:     func(record *core.Record) { record.Set("published", true) },
			update:      func(record *core.Record) { record.Set("description", "touched") },
			wantStarted: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			app, err := tests.NewTestApp(testDataDir)
			require.NoError(t, err)
			defer app.Cleanup()

			ensureWorkerManagerPublicationFields(t, app)
			canonify.RegisterCanonifyHooks(app)

			orgID, err := getOrgIDfromName(app)
			require.NoError(t, err)
			org, err := app.FindRecordById("organizations", orgID)
			require.NoError(t, err)
			org.Set("published", true)
			require.NoError(t, app.Save(org))

			runner := createWorkerManagerRunnerRecord(
				t,
				app,
				orgID,
				"gated-runner",
				"https://runner.example",
				false,
				false,
			)
			testCase.prepare(runner)
			require.NoError(t, app.Save(runner))
			// Reload so Original() carries the persisted pre-update state, as it
			// does for records fetched by a request or hook at runtime.
			runner, err = app.FindRecordById("mobile_runners", runner.Id)
			require.NoError(t, err)

			origStartManager := startWorkerManagerFn
			t.Cleanup(func() {
				startWorkerManagerFn = origStartManager
			})

			calls := make(chan string, 4)
			startWorkerManagerFn = func(_ core.App, namespace, _ string, runnerURLs []string) {
				require.Equal(t, []string{"https://runner.example"}, runnerURLs)
				calls <- namespace
			}
			RegisterMobileRunnerWorkerManagerHooks(app)

			testCase.update(runner)
			require.NoError(t, app.Save(runner))

			if testCase.wantStarted {
				select {
				case namespace := <-calls:
					require.Equal(t, org.GetString("canonified_name"), namespace)
				case <-time.After(2 * time.Second):
					t.Fatal("expected worker manager start")
				}
				return
			}

			select {
			case namespace := <-calls:
				t.Fatalf("unexpected worker manager start for namespace %q", namespace)
			case <-time.After(200 * time.Millisecond):
			}
		})
	}
}

// A runner coming back online is written by the lifecycle heartbeat handler
// inside a transaction, so the start must survive PocketBase deferring the
// after-update hook until the transaction commits.
func TestRegisterMobileRunnerWorkerManagerHooks_OnlineAgainInsideTransaction(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	ensureWorkerManagerPublicationFields(t, app)
	canonify.RegisterCanonifyHooks(app)

	orgID, err := getOrgIDfromName(app)
	require.NoError(t, err)
	org, err := app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	org.Set("published", true)
	require.NoError(t, app.Save(org))

	runner := createWorkerManagerRunnerRecord(
		t,
		app,
		orgID,
		"offline-runner",
		"https://runner.example",
		true,
		false,
	)
	runner.Set("online", false)
	require.NoError(t, app.Save(runner))

	origStartManager := startWorkerManagerFn
	t.Cleanup(func() {
		startWorkerManagerFn = origStartManager
	})

	calls := make(chan string, 2)
	startWorkerManagerFn = func(_ core.App, namespace, _ string, runnerURLs []string) {
		require.Equal(t, []string{"https://runner.example"}, runnerURLs)
		calls <- namespace
	}
	RegisterMobileRunnerWorkerManagerHooks(app)

	require.NoError(t, app.RunInTransaction(func(txApp core.App) error {
		current, err := txApp.FindRecordById("mobile_runners", runner.Id)
		if err != nil {
			return err
		}
		current.Set("online", true)

		return txApp.Save(current)
	}))

	select {
	case namespace := <-calls:
		require.Equal(t, org.GetString("canonified_name"), namespace)
	case <-time.After(2 * time.Second):
		t.Fatal("expected worker manager start after transactional online update")
	}
}

func TestRegisterMobileRunnerWorkerManagerHooks_AdminRunnerReEnabledCoversAllNamespaces(
	t *testing.T,
) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	ensureWorkerManagerPublicationFields(t, app)
	canonify.RegisterCanonifyHooks(app)

	orgID, err := getOrgIDfromName(app)
	require.NoError(t, err)
	publishedOrg, err := app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	publishedOrg.Set("published", true)
	require.NoError(t, app.Save(publishedOrg))

	orgsColl, err := app.FindCollectionByNameOrId("organizations")
	require.NoError(t, err)
	unpublishedOrg := core.NewRecord(orgsColl)
	unpublishedOrg.Set("name", "Org B")
	unpublishedOrg.Set("canonified_name", "org-b")
	unpublishedOrg.Set("published", false)
	require.NoError(t, app.Save(unpublishedOrg))

	runner := createWorkerManagerRunnerRecord(
		t,
		app,
		orgID,
		"admin-runner",
		"https://admin-runner.example",
		false,
		true,
	)
	runner.Set("disabled", true)
	require.NoError(t, app.Save(runner))
	runner, err = app.FindRecordById("mobile_runners", runner.Id)
	require.NoError(t, err)

	origStartManager := startWorkerManagerFn
	t.Cleanup(func() {
		startWorkerManagerFn = origStartManager
	})

	calls := make(chan string, 8)
	startWorkerManagerFn = func(_ core.App, namespace, _ string, runnerURLs []string) {
		require.Equal(t, []string{"https://admin-runner.example"}, runnerURLs)
		calls <- namespace
	}
	RegisterMobileRunnerWorkerManagerHooks(app)

	runner.Set("disabled", false)
	require.NoError(t, app.Save(runner))

	allOrgs, err := listAllOrganizationRecords(app)
	require.NoError(t, err)
	expected := make([]string, 0, 1+len(allOrgs))
	expected = append(expected, "default")
	for _, record := range allOrgs {
		expected = append(expected, record.GetString("canonified_name"))
	}
	require.Contains(t, expected, publishedOrg.GetString("canonified_name"))
	require.Contains(t, expected, "org-b")

	got := make([]string, 0, len(expected))
	for range expected {
		select {
		case namespace := <-calls:
			got = append(got, namespace)
		case <-time.After(2 * time.Second):
			t.Fatalf("expected %d worker manager starts, got %v", len(expected), got)
		}
	}
	require.ElementsMatch(t, expected, got)

	// An unrelated write on an already startable admin runner starts nothing.
	runner, err = app.FindRecordById("mobile_runners", runner.Id)
	require.NoError(t, err)
	runner.Set("description", "touched")
	require.NoError(t, app.Save(runner))

	select {
	case namespace := <-calls:
		t.Fatalf("unexpected worker manager start for namespace %q", namespace)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestRegisterMobileRunnerWorkerManagerHooks_AdminRunnerStaysSkippedWhileOffline(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	ensureWorkerManagerPublicationFields(t, app)
	canonify.RegisterCanonifyHooks(app)

	orgID, err := getOrgIDfromName(app)
	require.NoError(t, err)
	org, err := app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	org.Set("published", true)
	require.NoError(t, app.Save(org))

	runner := createWorkerManagerRunnerRecord(
		t,
		app,
		orgID,
		"offline-admin-runner",
		"https://admin-runner.example",
		false,
		true,
	)
	runner.Set("online", false)
	runner.Set("disabled", true)
	require.NoError(t, app.Save(runner))
	runner, err = app.FindRecordById("mobile_runners", runner.Id)
	require.NoError(t, err)

	origStartManager := startWorkerManagerFn
	t.Cleanup(func() {
		startWorkerManagerFn = origStartManager
	})

	calls := make(chan string, 4)
	startWorkerManagerFn = func(_ core.App, namespace, _ string, _ []string) {
		calls <- namespace
	}
	RegisterMobileRunnerWorkerManagerHooks(app)

	runner.Set("disabled", false)
	require.NoError(t, app.Save(runner))

	select {
	case namespace := <-calls:
		t.Fatalf("unexpected worker manager start for namespace %q", namespace)
	case <-time.After(200 * time.Millisecond):
	}
}
