// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine/hooks"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

func stubEnsureNamespace(t *testing.T, ensureErr error) (<-chan string, <-chan string) {
	t.Helper()
	origEnsure := ensureNamespaceReadyFn
	origStart := startWorkersByNamespaceFn
	t.Cleanup(func() {
		ensureNamespaceReadyFn = origEnsure
		startWorkersByNamespaceFn = origStart
	})

	ensured := make(chan string, 1)
	ensureNamespaceReadyFn = func(namespace string) error {
		ensured <- namespace
		return ensureErr
	}
	started := make(chan string, 1)
	startWorkersByNamespaceFn = func(_ core.App, namespace string) {
		started <- namespace
	}
	return ensured, started
}

func TestEnsureNamespaceAndWorkersCreatesNamespace(t *testing.T) {
	ensured, started := stubEnsureNamespace(t, nil)

	ensureNamespaceAndWorkers(nil, "tenant")

	require.Equal(t, "tenant", <-ensured)
	select {
	case ns := <-started:
		require.Equal(t, "tenant", ns)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for workers start")
	}
}

func TestEnsureNamespaceAndWorkersStartsWorkersForExistingNamespace(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	ensured, started := stubEnsureNamespace(t, nil)

	registerOrganizationNamespaceHooks(app)

	// Rename an organization to a namespace that already exists in Temporal:
	// EnsureNamespaceReady succeeds without registering and workers must start.
	record := core.NewRecord(core.NewBaseCollection("organizations"))
	record.Id = "org1"
	record.Set("canonified_name", "old-org")
	require.NoError(t, record.PostScan())
	record.Set("canonified_name", "existing-org")
	event := &core.RecordEvent{App: app}
	event.Record = record

	err := app.OnRecordAfterUpdateSuccess("organizations").Trigger(
		event,
		func(_ *core.RecordEvent) error { return nil },
	)
	require.NoError(t, err)
	require.Equal(t, "existing-org", <-ensured)
	select {
	case ns := <-started:
		require.Equal(t, "existing-org", ns)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for workers start")
	}
}

func TestEnsureNamespaceAndWorkersSkipsWhenNamespaceNotReady(t *testing.T) {
	_, started := stubEnsureNamespace(t, errors.New("not ready"))

	ensureNamespaceAndWorkers(nil, "tenant")

	select {
	case <-started:
		t.Fatal("workers must not start when the namespace is not ready")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestEnsureNamespaceAndWorkersSkipsWhenTemporalWorkersDisabled(t *testing.T) {
	t.Setenv(hooks.TemporalWorkersDisabledEnv, "1")

	origEnsure := ensureNamespaceReadyFn
	origStart := startWorkersByNamespaceFn
	t.Cleanup(func() {
		ensureNamespaceReadyFn = origEnsure
		startWorkersByNamespaceFn = origStart
	})

	ensureNamespaceReadyFn = func(_ string) error {
		require.Fail(t, "ensureNamespaceReady should not be called")
		return nil
	}
	startWorkersByNamespaceFn = func(_ core.App, _ string) {
		require.Fail(t, "startWorkersByNamespace should not be called")
	}

	ensureNamespaceAndWorkers(nil, "tenant")
}

func TestOrganizationNamespaceHooksAfterCreate(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})

	origEnsure := ensureNamespaceAndWorkersFn
	t.Cleanup(func() {
		ensureNamespaceAndWorkersFn = origEnsure
	})

	var ensured string
	ensureNamespaceAndWorkersFn = func(_ core.App, namespace string) {
		ensured = namespace
	}
	registerOrganizationNamespaceHooks(app)

	collection := core.NewBaseCollection("organizations")
	record := core.NewRecord(collection)
	record.Set("canonified_name", "org-1")
	event := &core.RecordEvent{App: app}
	event.Record = record

	err := app.OnRecordAfterCreateSuccess("organizations").Trigger(
		event,
		func(_ *core.RecordEvent) error { return nil },
	)
	require.NoError(t, err)
	require.Equal(t, "org-1", ensured)
}

func TestOrganizationNamespaceHooksCreateDefaults(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	registerOrganizationNamespaceHooks(app)

	collection := core.NewBaseCollection("organizations")
	record := core.NewRecord(collection)

	event := &core.RecordEvent{App: app}
	event.Record = record

	err := app.OnRecordCreate("organizations").Trigger(
		event,
		func(_ *core.RecordEvent) error { return nil },
	)
	require.NoError(t, err)
	require.Equal(t, defaultMaxPipelinesInQueue, record.GetInt("max_pipelines_in_queue"))
}

func TestOrganizationProtectedFieldsHooks_RevertsMaxPipelinesInQueueForUser(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	registerOrganizationProtectedFieldsHooks(app)

	org := loadOrgWithMaxPipelines(t, app, 3)

	userAuth := core.NewRecord(mustFindCollection(t, app, "users"))
	event := newOrganizationUpdateRequestEvent(app, org, userAuth)
	event.Record.Set("max_pipelines_in_queue", 99)

	err = app.OnRecordUpdateRequest("organizations").Trigger(
		event,
		func(_ *core.RecordRequestEvent) error { return nil },
	)
	require.NoError(t, err)
	require.Equal(t, 3, event.Record.GetInt("max_pipelines_in_queue"))
}

func TestOrganizationProtectedFieldsHooks_AllowsMaxPipelinesInQueueForSuperuser(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	registerOrganizationProtectedFieldsHooks(app)

	org := loadOrgWithMaxPipelines(t, app, 3)

	superuserAuth := core.NewRecord(mustFindCollection(t, app, core.CollectionNameSuperusers))
	event := newOrganizationUpdateRequestEvent(app, org, superuserAuth)
	event.Record.Set("max_pipelines_in_queue", 99)

	err = app.OnRecordUpdateRequest("organizations").Trigger(
		event,
		func(_ *core.RecordRequestEvent) error { return nil },
	)
	require.NoError(t, err)
	require.Equal(t, 99, event.Record.GetInt("max_pipelines_in_queue"))
}

func TestOrganizationProtectedFieldsHooks_RevertsPublishedForUser(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	ensureOrganizationPublicationFields(t, app)
	registerOrganizationProtectedFieldsHooks(app)

	org := loadOrgWithPublished(t, app, false)

	userAuth := core.NewRecord(mustFindCollection(t, app, "users"))
	event := newOrganizationUpdateRequestEvent(app, org, userAuth)
	event.Record.Set("published", true)

	err = app.OnRecordUpdateRequest("organizations").Trigger(
		event,
		func(_ *core.RecordRequestEvent) error { return nil },
	)
	require.NoError(t, err)
	require.False(t, event.Record.GetBool("published"))
}

func TestOrganizationProtectedFieldsHooks_AllowsPublishedForSuperuser(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	ensureOrganizationPublicationFields(t, app)
	registerOrganizationProtectedFieldsHooks(app)

	org := loadOrgWithPublished(t, app, false)

	superuserAuth := core.NewRecord(mustFindCollection(t, app, core.CollectionNameSuperusers))
	event := newOrganizationUpdateRequestEvent(app, org, superuserAuth)
	event.Record.Set("published", true)

	err = app.OnRecordUpdateRequest("organizations").Trigger(
		event,
		func(_ *core.RecordRequestEvent) error { return nil },
	)
	require.NoError(t, err)
	require.True(t, event.Record.GetBool("published"))
}

func loadOrgWithPublished(t testing.TB, app core.App, value bool) *core.Record {
	t.Helper()

	orgID, err := getOrgIDfromName(app)
	require.NoError(t, err)

	org, err := app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	org.Set("published", value)
	require.NoError(t, app.Save(org))

	org, err = app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	require.Equal(t, value, org.GetBool("published"))
	return org
}

func loadOrgWithMaxPipelines(t testing.TB, app core.App, value int) *core.Record {
	t.Helper()

	orgID, err := getOrgIDfromName(app)
	require.NoError(t, err)

	org, err := app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	org.Set("max_pipelines_in_queue", value)
	require.NoError(t, app.Save(org))

	org, err = app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	require.Equal(t, value, org.GetInt("max_pipelines_in_queue"))
	return org
}

func mustFindCollection(t testing.TB, app core.App, name string) *core.Collection {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(name)
	require.NoError(t, err)
	return collection
}

func newOrganizationUpdateRequestEvent(
	app core.App,
	org *core.Record,
	auth *core.Record,
) *core.RecordRequestEvent {
	requestEvent := &core.RequestEvent{App: app}
	requestEvent.Auth = auth

	event := &core.RecordRequestEvent{
		RequestEvent: requestEvent,
		Record:       org,
	}
	event.Collection = org.Collection()
	return event
}

func TestOrganizationPublicationHooks_PublishesOrgWhenWalletPublished(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	ensureOrganizationPublicationFields(t, app)
	canonify.RegisterCanonifyHooks(app)
	RegisterOrganizationPublicationHooks(app)

	orgID, err := getOrgIDfromName(app)
	require.NoError(t, err)

	org, err := app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	require.False(t, org.GetBool("published"))

	wallet := createOrganizationPublicationWallet(t, app, orgID)
	require.NoError(t, app.Save(wallet))

	org, err = app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	require.True(t, org.GetBool("published"))
}

func TestOrganizationPublicationHooks_UnpublishesOrgWhenLastPublicEntityUnpublished(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	ensureOrganizationPublicationFields(t, app)
	canonify.RegisterCanonifyHooks(app)
	RegisterOrganizationPublicationHooks(app)

	orgID, err := getOrgIDfromName(app)
	require.NoError(t, err)

	wallet := createOrganizationPublicationWallet(t, app, orgID)
	require.NoError(t, app.Save(wallet))

	wallet.Set("published", false)
	require.NoError(t, app.Save(wallet))

	org, err := app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	require.False(t, org.GetBool("published"))
}

func TestOrganizationPublicationHooks_KeepsOrgPublishedWhileAnotherEntityPublic(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	ensureOrganizationPublicationFields(t, app)
	canonify.RegisterCanonifyHooks(app)
	RegisterOrganizationPublicationHooks(app)

	orgID, err := getOrgIDfromName(app)
	require.NoError(t, err)

	wallet := createOrganizationPublicationWallet(t, app, orgID)
	require.NoError(t, app.Save(wallet))

	pipeline := createTestPipelineRecord(t, app)
	pipeline.Set("published", true)
	require.NoError(t, app.Save(pipeline))

	wallet.Set("published", false)
	require.NoError(t, app.Save(wallet))

	org, err := app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	require.True(t, org.GetBool("published"))

	pipeline.Set("published", false)
	require.NoError(t, app.Save(pipeline))

	org, err = app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	require.False(t, org.GetBool("published"))
}

func TestOrganizationPublicationHooks_BlocksManualUnpublishWithPublicEntity(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	ensureOrganizationPublicationFields(t, app)
	canonify.RegisterCanonifyHooks(app)
	RegisterOrganizationPublicationHooks(app)

	orgID, err := getOrgIDfromName(app)
	require.NoError(t, err)

	wallet := createOrganizationPublicationWallet(t, app, orgID)
	require.NoError(t, app.Save(wallet))

	org, err := app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	org.Set("published", false)

	err = app.Save(org)
	require.Error(t, err)

	var apiErr *router.ApiError
	require.True(t, errors.As(err, &apiErr))
	require.Equal(t, http.StatusBadRequest, apiErr.Status)
	require.Contains(t, apiErr.Message, "Organization cannot be unpublished")

	org, err = app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	require.True(t, org.GetBool("published"))
}

func createOrganizationPublicationWallet(
	t testing.TB,
	app core.App,
	orgID string,
) *core.Record {
	t.Helper()

	walletsColl, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	wallet := core.NewRecord(walletsColl)
	wallet.Set("owner", orgID)
	wallet.Set("name", "hook-test-wallet")
	wallet.Set("published", true)

	return wallet
}

func ensureOrganizationPublicationFields(t testing.TB, app *tests.TestApp) {
	t.Helper()

	organizations, err := app.FindCollectionByNameOrId("organizations")
	require.NoError(t, err)
	if organizations.Fields.GetByName("published") == nil {
		organizations.Fields.Add(&core.BoolField{Name: "published"})
	}
	require.NoError(t, app.Save(organizations))
}
