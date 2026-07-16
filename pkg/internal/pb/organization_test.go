// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
)

type fakeNamespaceClient struct {
	describeErrs  []error
	describeCalls int
}

func (f *fakeNamespaceClient) Register(
	_ context.Context,
	_ *workflowservice.RegisterNamespaceRequest,
) error {
	return nil
}

func (f *fakeNamespaceClient) Describe(
	_ context.Context,
	_ string,
) (*workflowservice.DescribeNamespaceResponse, error) {
	f.describeCalls++
	if len(f.describeErrs) > 0 {
		err := f.describeErrs[0]
		f.describeErrs = f.describeErrs[1:]
		return nil, err
	}
	return &workflowservice.DescribeNamespaceResponse{}, nil
}

func (f *fakeNamespaceClient) Update(
	_ context.Context,
	_ *workflowservice.UpdateNamespaceRequest,
) error {
	return nil
}

func (f *fakeNamespaceClient) Close() {}

func TestWaitForNamespaceReadyImmediateSuccess(t *testing.T) {
	client := &fakeNamespaceClient{}

	err := waitForNamespaceReady(client, "default", time.Second)
	require.NoError(t, err)
	require.Equal(t, 1, client.describeCalls)
}

func TestWaitForNamespaceReadyRetriesThenSucceeds(t *testing.T) {
	client := &fakeNamespaceClient{describeErrs: []error{errors.New("transient")}}

	err := waitForNamespaceReady(client, "default", 3*time.Second)
	require.NoError(t, err)
	require.GreaterOrEqual(t, client.describeCalls, 2)
}

func TestWaitForNamespaceReadyTimeout(t *testing.T) {
	client := &fakeNamespaceClient{describeErrs: []error{errors.New("still failing")}}

	err := waitForNamespaceReady(client, "default", -time.Second)
	require.Error(t, err)
	require.Equal(t, 1, client.describeCalls)
}

func TestEnsureNamespaceAndWorkersCreatesNamespace(t *testing.T) {
	origClient := newNamespaceClient
	origWait := waitForNamespaceReadyFn
	origStart := startWorkersByNamespaceFn
	t.Cleanup(func() {
		newNamespaceClient = origClient
		waitForNamespaceReadyFn = origWait
		startWorkersByNamespaceFn = origStart
	})

	mockClient := mocks.NewNamespaceClient(t)
	mockClient.
		On("Describe", mock.Anything, "tenant").
		Return((*workflowservice.DescribeNamespaceResponse)(nil), &serviceerror.NamespaceNotFound{}).
		Once()
	mockClient.On("Register", mock.Anything, mock.Anything).Return(nil).Once()
	mockClient.On("Close").Return()

	newNamespaceClient = func(_ client.Options) (client.NamespaceClient, error) {
		return mockClient, nil
	}

	waitCalled := make(chan struct{}, 1)
	waitForNamespaceReadyFn = func(_ client.NamespaceClient, namespace string, _ time.Duration) error {
		require.Equal(t, "tenant", namespace)
		waitCalled <- struct{}{}
		return nil
	}

	started := make(chan string, 1)
	startWorkersByNamespaceFn = func(namespace string) {
		started <- namespace
	}

	ensureNamespaceAndWorkers("tenant")

	select {
	case <-waitCalled:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for namespace readiness call")
	}

	select {
	case ns := <-started:
		require.Equal(t, "tenant", ns)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for workers start")
	}
}

func TestEnsureNamespaceAndWorkersSkipsExisting(t *testing.T) {
	origClient := newNamespaceClient
	origWait := waitForNamespaceReadyFn
	origStart := startWorkersByNamespaceFn
	t.Cleanup(func() {
		newNamespaceClient = origClient
		waitForNamespaceReadyFn = origWait
		startWorkersByNamespaceFn = origStart
	})

	mockClient := mocks.NewNamespaceClient(t)
	mockClient.
		On("Describe", mock.Anything, "tenant").
		Return(&workflowservice.DescribeNamespaceResponse{}, nil).
		Once()
	mockClient.On("Close").Return()

	newNamespaceClient = func(_ client.Options) (client.NamespaceClient, error) {
		return mockClient, nil
	}

	waitForNamespaceReadyFn = func(_ client.NamespaceClient, _ string, _ time.Duration) error {
		require.Fail(t, "waitForNamespaceReady should not be called")
		return nil
	}

	startWorkersByNamespaceFn = func(_ string) {
		require.Fail(t, "startWorkersByNamespace should not be called")
	}

	ensureNamespaceAndWorkers("tenant")
}

func TestHookNamespaceOrgsAfterCreate(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})

	origEnsure := ensureNamespaceAndWorkersFn
	origStartManager := startWorkerManagerFn
	origAdminRunnerURLs := adminRunnerURLsFn
	t.Cleanup(func() {
		ensureNamespaceAndWorkersFn = origEnsure
		startWorkerManagerFn = origStartManager
		adminRunnerURLsFn = origAdminRunnerURLs
	})

	var ensured string
	ensureNamespaceAndWorkersFn = func(namespace string) {
		ensured = namespace
	}

	adminRunnerURLsFn = func(_ core.App) ([]string, error) {
		return []string{"https://admin.runner"}, nil
	}

	var started struct {
		namespace    string
		oldNamespace string
		runnerURLs   []string
	}
	startWorkerManagerFn = func(_ core.App, namespace, oldNamespace string, runnerURLs []string) {
		started.namespace = namespace
		started.oldNamespace = oldNamespace
		started.runnerURLs = runnerURLs
	}

	HookNamespaceOrgs(app)

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
	require.Equal(t, "org-1", started.namespace)
	require.Equal(t, "", started.oldNamespace)
	require.Equal(t, []string{"https://admin.runner"}, started.runnerURLs)
}

func TestHookNamespaceOrgsCreateDefaults(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	HookNamespaceOrgs(app)

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

func TestHookOrganizations_OrganizationPublishesToNonAdminRunners(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	ensureWorkerManagerPublicationFields(t, app)
	canonify.RegisterCanonifyHooks(app)

	orgID, err := getOrgIDfromName(app)
	require.NoError(t, err)
	org, err := app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	org.Set("published", false)
	require.NoError(t, app.Save(org))

	createWorkerManagerRunnerRecord(
		t,
		app,
		orgID,
		"private-admin",
		"https://admin.example",
		false,
		true,
	)
	createWorkerManagerRunnerRecord(
		t,
		app,
		orgID,
		"public-user",
		"https://public.example",
		true,
		false,
	)

	origStartManager := startWorkerManagerFn
	t.Cleanup(func() {
		startWorkerManagerFn = origStartManager
	})

	type call struct {
		namespace  string
		runnerURLs []string
	}
	calls := make(chan call, 1)
	startWorkerManagerFn = func(_ core.App, namespace, oldNamespace string, runnerURLs []string) {
		require.Empty(t, oldNamespace)
		calls <- call{namespace: namespace, runnerURLs: runnerURLs}
	}
	HookOrganizations(app)

	org.Set("published", true)
	require.NoError(t, app.Save(org))

	select {
	case got := <-calls:
		require.Equal(t, org.GetString("canonified_name"), got.namespace)
		require.Equal(t, []string{"https://public.example"}, got.runnerURLs)
	default:
		t.Fatal("expected worker manager call")
	}
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

func ensureWorkerManagerPublicationFields(t testing.TB, app *tests.TestApp) {
	t.Helper()

	orgs, err := app.FindCollectionByNameOrId("organizations")
	require.NoError(t, err)
	if orgs.Fields.GetByName("published") == nil {
		orgs.Fields.Add(&core.BoolField{Name: "published"})
	}
	require.NoError(t, app.Save(orgs))

	runners, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	if runners.Fields.GetByName("published") == nil {
		runners.Fields.Add(&core.BoolField{Name: "published"})
	}
	if runners.Fields.GetByName("admin_managed") == nil {
		runners.Fields.Add(&core.BoolField{Name: "admin_managed"})
	}
	require.NoError(t, app.Save(runners))
}

func createWorkerManagerRunnerRecord(
	t testing.TB,
	app core.App,
	orgID string,
	name string,
	ip string,
	published bool,
	adminManaged bool,
) *core.Record {
	t.Helper()

	runners, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)

	record := core.NewRecord(runners)
	record.Set("owner", orgID)
	record.Set("name", name)
	record.Set("ip", ip)
	record.Set("type", "android_emulator")
	record.Set("published", published)
	record.Set("admin_managed", adminManaged)
	require.NoError(t, app.Save(record))

	return record
}
