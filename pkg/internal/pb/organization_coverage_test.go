// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"errors"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
)

const covWESecondOrgID = "3u4982xn6ah0433"

type covWEWorkerManagerCall struct {
	namespace    string
	oldNamespace string
	runnerURLs   []string
}

// covWEStubOrgWorkerFns replaces the worker side effects of organization hooks.
func covWEStubOrgWorkerFns(
	t *testing.T,
	runnerURLsErr error,
) (*[]string, *[]covWEWorkerManagerCall) {
	t.Helper()
	origEnsure := ensureNamespaceAndWorkersFn
	origStartManager := startWorkerManagerFn
	origAdminRunnerURLs := adminRunnerURLsFn
	t.Cleanup(func() {
		ensureNamespaceAndWorkersFn = origEnsure
		startWorkerManagerFn = origStartManager
		adminRunnerURLsFn = origAdminRunnerURLs
	})

	ensured := &[]string{}
	calls := &[]covWEWorkerManagerCall{}
	ensureNamespaceAndWorkersFn = func(_ core.App, namespace string) {
		*ensured = append(*ensured, namespace)
	}
	adminRunnerURLsFn = func(_ core.App) ([]string, error) {
		if runnerURLsErr != nil {
			return nil, runnerURLsErr
		}
		return []string{"https://admin.runner"}, nil
	}
	startWorkerManagerFn = func(namespace, oldNamespace string, runnerURLs []string) {
		*calls = append(*calls, covWEWorkerManagerCall{namespace, oldNamespace, runnerURLs})
	}
	return ensured, calls
}

// covWEOrgCollection returns an organizations collection declaring the fields hooks read.
func covWEOrgCollection() *core.Collection {
	collection := core.NewBaseCollection("organizations")
	collection.Fields.Add(
		&core.TextField{Name: "canonified_name"},
		&core.BoolField{Name: "published"},
	)
	return collection
}

// covWEOrgRecord builds an organizations record whose original state has oldName.
func covWEOrgRecord(t *testing.T, oldName, newName string) *core.Record {
	t.Helper()
	record := core.NewRecord(covWEOrgCollection())
	record.Id = "covweorg0000001"
	record.Set("canonified_name", oldName)
	require.NoError(t, record.PostScan())
	record.Set("canonified_name", newName)
	return record
}

func TestCovWEOrganizationRenameMovesWorkers(t *testing.T) {
	tests := []struct {
		name          string
		oldName       string
		newName       string
		runnerURLsErr error
		wantErr       bool
		wantEnsured   []string
		wantCalls     []covWEWorkerManagerCall
	}{
		{
			name:        "rename moves workers and worker manager",
			oldName:     "old-org",
			newName:     "new-org",
			wantEnsured: []string{"new-org"},
			wantCalls: []covWEWorkerManagerCall{
				{
					namespace:    "new-org",
					oldNamespace: "old-org",
					runnerURLs:   []string{"https://admin.runner"},
				},
			},
		},
		{
			name:        "unchanged name is a no-op",
			oldName:     "same-org",
			newName:     "same-org",
			wantEnsured: []string{},
			wantCalls:   []covWEWorkerManagerCall{},
		},
		{
			name:        "cleared name is a no-op",
			oldName:     "old-org",
			newName:     "",
			wantEnsured: []string{},
			wantCalls:   []covWEWorkerManagerCall{},
		},
		{
			name:          "runner lookup failure aborts before worker manager start",
			oldName:       "old-org",
			newName:       "new-org",
			runnerURLsErr: errors.New("runners unavailable"),
			wantErr:       true,
			wantEnsured:   []string{"new-org"},
			wantCalls:     []covWEWorkerManagerCall{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
			ensured, calls := covWEStubOrgWorkerFns(t, tc.runnerURLsErr)
			HookNamespaceOrgs(app)

			event := &core.RecordEvent{App: app}
			event.Record = covWEOrgRecord(t, tc.oldName, tc.newName)
			err := app.OnRecordAfterUpdateSuccess("organizations").Trigger(
				event,
				func(_ *core.RecordEvent) error { return nil },
			)

			if tc.wantErr {
				require.ErrorIs(t, err, tc.runnerURLsErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.wantEnsured, *ensured)
			assert.Equal(t, tc.wantCalls, *calls)
		})
	}
}

func TestCovWEOrganizationCreateFailsWhenRunnerLookupFails(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	lookupErr := errors.New("runners unavailable")
	ensured, calls := covWEStubOrgWorkerFns(t, lookupErr)
	HookNamespaceOrgs(app)

	record := core.NewRecord(core.NewBaseCollection("organizations"))
	record.Set("canonified_name", "org-1")
	event := &core.RecordEvent{App: app}
	event.Record = record
	err := app.OnRecordAfterCreateSuccess("organizations").Trigger(
		event,
		func(_ *core.RecordEvent) error { return nil },
	)

	require.ErrorIs(t, err, lookupErr)
	assert.Equal(t, []string{"org-1"}, *ensured)
	assert.Empty(t, *calls)
}

func TestCovWEOrganizationWorkerManagerPublicationHookSkips(t *testing.T) {
	tests := []struct {
		name         string
		wasPublished bool
		published    bool
		namespace    string
	}{
		{name: "already published", wasPublished: true, published: true, namespace: "org"},
		{name: "unpublished", wasPublished: false, published: false, namespace: "org"},
		{name: "published without namespace", wasPublished: false, published: true, namespace: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
			_, calls := covWEStubOrgWorkerFns(t, nil)
			registerOrganizationWorkerManagerPublicationHooks(app)

			record := core.NewRecord(covWEOrgCollection())
			record.Id = "covweorg0000002"
			record.Set("published", tc.wasPublished)
			require.NoError(t, record.PostScan())
			record.Set("published", tc.published)
			record.Set("canonified_name", tc.namespace)
			event := &core.RecordEvent{App: app}
			event.Record = record

			require.NoError(t, app.OnRecordAfterUpdateSuccess("organizations").Trigger(
				event,
				func(_ *core.RecordEvent) error { return nil },
			))
			assert.Empty(t, *calls)
		})
	}
}

func covWEPublicationApp(t *testing.T) (*tests.TestApp, string) {
	t.Helper()
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)
	ensureOrganizationPublicationFields(t, app)
	canonify.RegisterCanonifyHooks(app)
	RegisterOrganizationPublicationHooks(app)
	orgID, err := getOrgIDfromName(app)
	require.NoError(t, err)
	return app, orgID
}

func covWEOrgPublished(t *testing.T, app core.App, orgID string) bool {
	t.Helper()
	org, err := app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	return org.GetBool("published")
}

func TestCovWEOrganizationUnpublishedWhenLastPublicWalletDeleted(t *testing.T) {
	app, orgID := covWEPublicationApp(t)
	wallet := createOrganizationPublicationWallet(t, app, orgID)
	require.NoError(t, app.Save(wallet))
	require.True(t, covWEOrgPublished(t, app, orgID))

	require.NoError(t, app.Delete(wallet))

	assert.False(t, covWEOrgPublished(t, app, orgID))
}

func TestCovWEOrganizationPublicationFollowsWalletOwnerChange(t *testing.T) {
	app, orgID := covWEPublicationApp(t)
	wallet := createOrganizationPublicationWallet(t, app, orgID)
	require.NoError(t, app.Save(wallet))
	require.True(t, covWEOrgPublished(t, app, orgID))

	wallet, err := app.FindRecordById("wallets", wallet.Id)
	require.NoError(t, err)
	wallet.Set("owner", covWESecondOrgID)
	require.NoError(t, app.Save(wallet))

	assert.False(t, covWEOrgPublished(t, app, orgID), "previous owner loses its only public record")
	assert.True(t, covWEOrgPublished(t, app, covWESecondOrgID), "new owner becomes published")
}

func TestCovWEOrganizationStaysPublishedOnUnrelatedUpdate(t *testing.T) {
	app, orgID := covWEPublicationApp(t)
	wallet := createOrganizationPublicationWallet(t, app, orgID)
	require.NoError(t, app.Save(wallet))

	org, err := app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	require.True(t, org.GetBool("published"))
	org.Set("max_pipelines_in_queue", org.GetInt("max_pipelines_in_queue")+1)
	require.NoError(t, app.Save(org))

	assert.True(t, covWEOrgPublished(t, app, orgID))
}

func TestCovWEOrganizationManualUnpublishAllowedWithoutPublicRecords(t *testing.T) {
	app, orgID := covWEPublicationApp(t)
	org, err := app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	org.Set("published", true)
	require.NoError(t, app.Save(org))

	org.Set("published", false)
	require.NoError(t, app.Save(org))

	assert.False(t, covWEOrgPublished(t, app, orgID))
}

func TestCovWEEnsureNamespaceAndWorkersFailures(t *testing.T) {
	notFound := &serviceerror.NamespaceNotFound{}
	tests := []struct {
		name        string
		clientErr   error
		describeErr error
		registerErr error
		waitErr     error
		wantWait    bool
	}{
		{name: "client creation fails", clientErr: errors.New("dial failed")},
		{
			name:        "namespace registration fails",
			describeErr: notFound,
			registerErr: errors.New("register denied"),
		},
		{
			name:        "describe fails with other error",
			describeErr: errors.New("unavailable"),
		},
		{
			name:        "namespace never becomes ready",
			describeErr: notFound,
			waitErr:     errors.New("still not ready"),
			wantWait:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			origClient := newNamespaceClient
			origWait := waitForNamespaceReadyFn
			origStart := startWorkersByNamespaceFn
			t.Cleanup(func() {
				newNamespaceClient = origClient
				waitForNamespaceReadyFn = origWait
				startWorkersByNamespaceFn = origStart
			})

			mockClient := mocks.NewNamespaceClient(t)
			if tc.clientErr == nil {
				mockClient.On("Describe", mock.Anything, "tenant").
					Return((*workflowservice.DescribeNamespaceResponse)(nil), tc.describeErr).
					Once()
				mockClient.On("Close").Return()
				if errors.Is(tc.describeErr, notFound) {
					mockClient.On("Register", mock.Anything, mock.Anything).
						Return(tc.registerErr).
						Once()
				}
			}
			newNamespaceClient = func(_ client.Options) (client.NamespaceClient, error) {
				if tc.clientErr != nil {
					return nil, tc.clientErr
				}
				return mockClient, nil
			}
			waitCalled := false
			waitForNamespaceReadyFn = func(_ client.NamespaceClient, _ string, _ time.Duration) error {
				waitCalled = true
				return tc.waitErr
			}
			started := make(chan string, 1)
			startWorkersByNamespaceFn = func(_ core.App, namespace string) {
				started <- namespace
			}

			ensureNamespaceAndWorkers(nil, "tenant")

			assert.Equal(t, tc.wantWait, waitCalled)
			select {
			case ns := <-started:
				t.Fatalf("workers unexpectedly started for %s", ns)
			case <-time.After(20 * time.Millisecond):
			}
		})
	}
}
