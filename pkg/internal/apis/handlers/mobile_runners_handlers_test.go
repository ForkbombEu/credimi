// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func setupMobileRunnerApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)

	canonify.RegisterCanonifyHooks(app)
	MobileRunnersTemporalInternalRoutes.Add(app)

	return app
}

func TestGetMobileRunner(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:           "missing runner_identifier parameter",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner",
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"runner_identifier"`,
				`"runner_identifier is required"`,
			},
			TestAppFactory: setupMobileRunnerApp,
		},
		{
			Name:           "nonexistent runner identifier",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner?runner_identifier=does-not-exist",
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"mobile runner not found"`,
			},
			TestAppFactory: setupMobileRunnerApp,
		},
		{
			Name:           "valid runner identifier",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner?runner_identifier=usera-s-organization/test-runner",
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"runner_url"`,
				`"serial"`,
				`https://192.168.1.10:8050`,
				`SERIAL123`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupMobileRunnerApp(t)

				coll, err := app.FindCollectionByNameOrId("mobile_runners")
				require.NoError(t, err)

				record := core.NewRecord(coll)
				record.Set("owner", orgID)
				record.Set("serial", "SERIAL123")
				record.Set("ip", "https://192.168.1.10")
				record.Set("port", "8050")
				record.Set("name", "test-runner")

				require.NoError(t, app.Save(record))

				return app
			},
		},
		{
			Name:           "valid runner identifier with no port",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner?runner_identifier=usera-s-organization/no-port-runner",
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"runner_url"`,
				`"serial"`,
				`http://192.168.1.20`,
				`SERIAL999`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupMobileRunnerApp(t)

				coll, err := app.FindCollectionByNameOrId("mobile_runners")
				require.NoError(t, err)

				record := core.NewRecord(coll)
				record.Set("owner", orgID)
				record.Set("serial", "SERIAL999")
				record.Set("ip", "http://192.168.1.20")
				record.Set("name", "no-port-runner")

				require.NoError(t, app.Save(record))

				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestListMobileRunnerURLs(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:           "empty runners list",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner/list-urls",
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"runners":[]`,
			},
			TestAppFactory: setupMobileRunnerApp,
		},
		{
			Name:           "multiple runners",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner/list-urls",
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"runners"`,
				`http://192.168.1.10`,
				`https://192.168.1.11:9000`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupMobileRunnerApp(t)

				coll, err := app.FindCollectionByNameOrId("mobile_runners")
				require.NoError(t, err)

				// Runner 1
				r1 := core.NewRecord(coll)
				r1.Set("owner", orgID)
				r1.Set("serial", "SERIAL1")
				r1.Set("ip", "http://192.168.1.10")
				r1.Set("name", "runner-1")

				// Runner 2
				r2 := core.NewRecord(coll)
				r2.Set("owner", orgID)
				r2.Set("serial", "SERIAL2")
				r2.Set("ip", "https://192.168.1.11")
				r2.Set("port", "9000")
				r2.Set("name", "runner-2")

				require.NoError(t, app.Save(r1))
				require.NoError(t, app.Save(r2))

				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestGetMobileRunnerSemaphore(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:           "missing runner_identifier parameter",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner/semaphore",
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"runner_identifier"`,
				`"runner_identifier is required"`,
			},
			TestAppFactory: setupMobileRunnerApp,
		},
		{
			Name:           "nonexistent runner identifier",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner/semaphore?runner_identifier=does-not-exist",
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"mobile runner not found"`,
			},
			TestAppFactory: setupMobileRunnerApp,
		},
		{
			Name:           "semaphore not found for runner",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner/semaphore?runner_identifier=usera-s-organization/runner-without-semaphore",
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"runner semaphore not found"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupMobileRunnerApp(t)

				coll, err := app.FindCollectionByNameOrId("mobile_runners")
				require.NoError(t, err)

				record := core.NewRecord(coll)
				record.Set("owner", orgID)
				record.Set("serial", "SERIAL123")
				record.Set("ip", "https://192.168.1.10")
				record.Set("port", "8050")
				record.Set("name", "runner-without-semaphore")
				require.NoError(t, app.Save(record))

				originalQuery := queryMobileRunnerSemaphoreState
				queryMobileRunnerSemaphoreState = func(_ context.Context, _ string) (workflows.MobileRunnerSemaphoreStateView, error) {
					return workflows.MobileRunnerSemaphoreStateView{}, errSemaphoreNotFound
				}
				t.Cleanup(func() {
					queryMobileRunnerSemaphoreState = originalQuery
				})

				return app
			},
		},
		{
			Name:           "semaphore state returned",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner/semaphore?runner_identifier=usera-s-organization/test-semaphore-runner",
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"runner_id":"test-semaphore-runner"`,
				`"capacity":1`,
				`"queue_len":2`,
				`"in_use":true`,
				`"holder"`,
				`"lease_id":"lease-1"`,
				`"last_grant_at"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupMobileRunnerApp(t)

				coll, err := app.FindCollectionByNameOrId("mobile_runners")
				require.NoError(t, err)

				record := core.NewRecord(coll)
				record.Set("owner", orgID)
				record.Set("serial", "SERIAL321")
				record.Set("ip", "https://192.168.1.99")
				record.Set("port", "9000")
				record.Set("name", "test-semaphore-runner")
				require.NoError(t, app.Save(record))

				originalQuery := queryMobileRunnerSemaphoreState
				queryMobileRunnerSemaphoreState = func(_ context.Context, _ string) (workflows.MobileRunnerSemaphoreStateView, error) {
					lastGrant := time.Date(2026, 1, 29, 10, 0, 0, 0, time.UTC)
					holder := workflows.MobileRunnerSemaphoreHolder{LeaseID: "lease-1"}
					return workflows.MobileRunnerSemaphoreStateView{
						RunnerID:     "test-semaphore-runner",
						Capacity:     1,
						Holders:      []workflows.MobileRunnerSemaphoreHolder{holder},
						CurrentHolder: &holder,
						QueueLen:     2,
						LastGrantAt:  &lastGrant,
					}, nil
				}
				t.Cleanup(func() {
					queryMobileRunnerSemaphoreState = originalQuery
				})

				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
