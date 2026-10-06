// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func TestScoreboardDurationFormatting(t *testing.T) {
	scenarios := []struct {
		name        string
		duration    time.Duration
		set         bool
		wantString  string
		wantSeconds int
	}{
		{name: "unset", duration: time.Hour, set: false, wantString: "", wantSeconds: 0},
		{
			name:        "seconds",
			duration:    1600 * time.Millisecond,
			set:         true,
			wantString:  "2s",
			wantSeconds: 2,
		},
		{
			name:        "minutes",
			duration:    2*time.Minute + 5*time.Second,
			set:         true,
			wantString:  "2m5s",
			wantSeconds: 125,
		},
		{
			name:        "hours",
			duration:    26*time.Hour + 3*time.Minute + 4*time.Second,
			set:         true,
			wantString:  "26h3m4s",
			wantSeconds: 93784,
		},
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			assert.Equal(t, s.wantString, formatDurationString(s.duration, s.set))
			assert.Equal(t, s.wantSeconds, durationSeconds(s.duration, s.set))
		})
	}
}

func TestScoreboardUpdateMinDuration(t *testing.T) {
	scenarios := []struct {
		name    string
		exec    WorkflowExecution
		current time.Duration
		set     bool
		want    time.Duration
		wantSet bool
	}{
		{
			name:    "missing close time is ignored",
			exec:    WorkflowExecution{StartTime: "2026-01-01T00:00:00Z"},
			current: time.Minute,
			set:     true,
			want:    time.Minute,
			wantSet: true,
		},
		{
			name:    "unparseable time is ignored",
			exec:    WorkflowExecution{StartTime: "yesterday", CloseTime: "2026-01-01T00:00:10Z"},
			want:    0,
			wantSet: false,
		},
		{
			name: "first duration is recorded",
			exec: WorkflowExecution{
				StartTime: "2026-01-01T00:00:00Z",
				CloseTime: "2026-01-01T00:00:10Z",
			},
			want:    10 * time.Second,
			wantSet: true,
		},
		{
			name: "longer duration keeps minimum",
			exec: WorkflowExecution{
				StartTime: "2026-01-01T00:00:00Z",
				CloseTime: "2026-01-01T00:01:00Z",
			},
			current: 10 * time.Second,
			set:     true,
			want:    10 * time.Second,
			wantSet: true,
		},
		{
			name: "shorter duration replaces minimum",
			exec: WorkflowExecution{
				StartTime: "2026-01-01T00:00:00Z",
				CloseTime: "2026-01-01T00:00:03Z",
			},
			current: 10 * time.Second,
			set:     true,
			want:    3 * time.Second,
			wantSet: true,
		},
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			got, gotSet := s.current, s.set
			updateMinDuration(&s.exec, &got, &gotSet)
			assert.Equal(t, s.want, got)
			assert.Equal(t, s.wantSet, gotSet)
		})
	}
}

func TestScoreboardSmallHelpers(t *testing.T) {
	t.Run("extractFirstTwoParts drops the last segment", func(t *testing.T) {
		assert.Equal(t, "org/wallet", extractFirstTwoParts("org/wallet/1-0-0"))
		assert.Equal(t, "single", extractFirstTwoParts("single"))
	})

	t.Run("updateDateRange ignores empty start", func(t *testing.T) {
		first, last := "a", "b"
		updateDateRange("", &first, &last)
		assert.Equal(t, "a", first)
		assert.Equal(t, "b", last)
	})

	t.Run("getStringListFromAttrs", func(t *testing.T) {
		attrs := DecodedWorkflowSearchAttributes{
			"typed": []string{"a"},
			"mixed": []interface{}{"x", 1, "y"},
			"other": "scalar",
		}
		assert.Equal(t, []string{"a"}, getStringListFromAttrs(attrs, "typed"))
		assert.Equal(t, []string{"x", "y"}, getStringListFromAttrs(attrs, "mixed"))
		assert.Nil(t, getStringListFromAttrs(attrs, "other"))
		assert.Nil(t, getStringListFromAttrs(attrs, "missing"))
	})

	t.Run("pipelineRunTypeFromMap defaults to manual", func(t *testing.T) {
		exec := &WorkflowExecution{Execution: &WorkflowIdentifier{WorkflowID: "wf", RunID: "run"}}
		ref := workflowExecutionRef{WorkflowID: "wf", RunID: "run"}
		assert.Equal(t, pipelineinternal.RunTypeManual, pipelineRunTypeFromMap(nil, nil))
		assert.Equal(t, pipelineinternal.RunTypeManual, pipelineRunTypeFromMap(
			map[workflowExecutionRef]string{ref: "bogus"},
			exec,
		))
		assert.Equal(t, pipelineinternal.RunTypeScheduled, pipelineRunTypeFromMap(
			map[workflowExecutionRef]string{ref: pipelineinternal.RunTypeScheduled},
			exec,
		))
	})

	t.Run("empty inputs", func(t *testing.T) {
		stats, last := calculateStatsFromExecutions(nil, nil, nil, nil)
		assert.Nil(t, last)
		assert.Equal(t, 0, stats.TotalRuns)
		assert.Equal(t, []string{}, resolveDeviceTypes(nil, []string{"d"}, nil))
		assert.Empty(t, pipelineRunTypesForExecutions(nil, []*WorkflowExecution{{}}))
		video, screenshot, logs := getPipelineResultFromRecord(nil, nil)
		assert.Empty(t, video+screenshot+logs)
		assert.Equal(t, &LastExecutionDetails{}, extractEntityDetailsFromExecution(nil))
	})
}

func TestPipelineRunTypesForExecutionsIgnoresInvalidStoredTypes(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	pipeline := createPipelineRecord(t, app, orgID, "cov-run-types")
	createPipelineResultWithType(
		t,
		app,
		orgID,
		pipeline.Id,
		"wf-ci",
		"run-ci",
		pipelineinternal.RunTypeCI,
	)
	createPipelineResult(t, app, orgID, pipeline.Id, "wf-untyped", "run-untyped")

	runTypes := pipelineRunTypesForExecutions(app, []*WorkflowExecution{
		nil,
		{},
		{Execution: &WorkflowIdentifier{WorkflowID: "wf-ci", RunID: "run-ci"}},
		{Execution: &WorkflowIdentifier{WorkflowID: "wf-untyped", RunID: "run-untyped"}},
	})
	assert.Equal(t, map[workflowExecutionRef]string{
		{WorkflowID: "wf-ci", RunID: "run-ci"}: pipelineinternal.RunTypeCI,
	}, runTypes)
}

func TestGetOrgLogo(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://credimi.test"

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	assert.Empty(t, getOrgLogo(app, ""), "empty namespace")
	assert.Empty(t, getOrgLogo(app, "missing-org"), "unknown namespace")
	assert.Empty(t, getOrgLogo(app, "usera-s-organization"), "organization without logo")

	_, err = app.DB().
		NewQuery("UPDATE organizations SET logo = 'logo_abc.png' WHERE id = {:id}").
		Bind(dbx.Params{"id": orgID}).
		Execute()
	require.NoError(t, err)
	assert.Equal(
		t,
		"https://credimi.test/api/files/organizations/"+orgID+"/logo/logo_abc.png",
		getOrgLogo(app, "usera-s-organization"),
	)
}

func TestTruncateCollection(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	require.Error(t, truncateCollection(app, "no_such_collection"))
	require.NoError(t, truncateCollection(app, "pipeline_results"), "empty collection")

	pipeline := createPipelineRecord(t, app, orgID, "cov-truncate")
	createPipelineResult(t, app, orgID, pipeline.Id, "wf-1", "run-1")
	createPipelineResult(t, app, orgID, pipeline.Id, "wf-2", "run-2")

	require.NoError(t, truncateCollection(app, "pipeline_results"))
	total, err := app.CountRecords("pipeline_results")
	require.NoError(t, err)
	assert.Zero(t, total)
}

func TestSaveScoreboardResultsFailsWhenNothingIsSaved(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	published := createPipelineRecord(t, app, orgID, "cov-published")
	published.Set("published", true)
	require.NoError(t, app.Save(published))

	count, saveErrors := insertAggregatedResults(app, nil)
	assert.Zero(t, count)
	require.Len(t, saveErrors, 1)

	scenarios := []struct {
		name       string
		pipelineID string
		failSave   bool
		wantErrMsg string
	}{
		{
			name:       "unknown pipeline",
			pipelineID: "missingpipe1234",
			wantErrMsg: "failed to find pipeline record",
		},
		{
			name:       "cache save rejected",
			pipelineID: published.Id,
			failSave:   true,
			wantErrMsg: "save: boom",
		},
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			if s.failSave {
				app.OnRecordCreate("pipeline_scoreboard_cache").
					BindFunc(func(*core.RecordEvent) error {
						return errors.New("boom")
					})
			}
			_, err := runScoreboardActivity[ScoreboardSaveOutput](
				t,
				NewSaveScoreboardResultsActivity(app),
				workflows.AggregateScoreboardWorkflowOutput{
					AggregatedPipelines: []workflows.AggregatedPipelineStats{
						{PipelineID: s.pipelineID},
					},
				},
			)
			requireActivityErrorCode(t, err, errorcodes.DatabaseOperationFailed)
			assert.ErrorContains(t, err, s.wantErrMsg)

			total, err := app.CountRecords("pipeline_scoreboard_cache")
			require.NoError(t, err)
			assert.Zero(t, total)
		})
	}
}

func TestScoreboardExpandedRecord(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://credimi.test"
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	wallet := createWalletRecord(t, app, orgID, "cov-wallet")
	version, err := app.FindFirstRecordByData("wallet_versions", "wallet", wallet.Id)
	require.NoError(t, err)
	private := createPipelineRecord(t, app, orgID, "cov-private")

	t.Run("empty id", func(t *testing.T) {
		entity, err := scoreboardExpandedRecord(app, "wallets", "")
		require.NoError(t, err)
		assert.Nil(t, entity)
	})

	t.Run("missing record", func(t *testing.T) {
		_, err := scoreboardExpandedRecord(app, "wallets", "missingwallet12")
		require.ErrorContains(t, err, "find wallets missingwallet12")
	})

	t.Run("unpublished record is hidden", func(t *testing.T) {
		entity, err := scoreboardExpandedRecord(app, "pipelines", private.Id)
		require.NoError(t, err)
		assert.Nil(t, entity)
	})

	t.Run("wallet version inherits wallet visibility", func(t *testing.T) {
		entity, err := scoreboardExpandedRecord(app, "wallet_versions", version.Id)
		require.NoError(t, err)
		require.NotNil(t, entity)
		assert.Equal(t, "1.0.0", entity.Tag)
		assert.Equal(t, wallet.Id, entity.Wallet)
		assert.Equal(t, "usera-s-organization/cov-wallet/1-0-0", entity.CanonifiedPath)

		wallet.Set("published", false)
		require.NoError(t, app.Save(wallet))
		entity, err = scoreboardExpandedRecord(app, "wallet_versions", version.Id)
		require.NoError(t, err)
		assert.Nil(t, entity)
	})

	t.Run("missing device", func(t *testing.T) {
		_, err := scoreboardExpandedDevices(app, []string{"missingdevice12"})
		require.ErrorContains(t, err, "find mobile device missingdevice12")
	})

	t.Run("missing latest execution", func(t *testing.T) {
		cache, err := app.FindCollectionByNameOrId("pipeline_scoreboard_cache")
		require.NoError(t, err)
		record := core.NewRecord(cache)
		record.Set("latest_execution", "missingresult12")
		_, err = buildScoreboardExpandedData(app, record)
		require.ErrorContains(t, err, "find latest execution")
	})
}

func TestScoreboardLogoURL(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://credimi.test"

	coll, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	scenarios := []struct {
		name    string
		logoURL string
		logo    string
		want    string
	}{
		{
			name:    "explicit logo url wins",
			logoURL: "https://cdn.test/l.png",
			logo:    "x.png",
			want:    "https://cdn.test/l.png",
		},
		{
			name: "uploaded logo",
			logo: "x.png",
			want: "https://credimi.test/api/files/wallets/rec123/x.png",
		},
		{name: "no logo", want: ""},
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			record := core.NewRecord(coll)
			record.Id = "rec123"
			record.Load(map[string]any{"logo_url": s.logoURL, "logo": s.logo})
			assert.Equal(t, s.want, scoreboardLogoURL(app, "wallets", record))
		})
	}
}

func TestNamespaceScoreboardAndDetailsTemporalFailures(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()
	ctx := context.Background()

	t.Run("no pipelines needs no temporal client", func(t *testing.T) {
		orig := pipelineResultsTemporalClient
		t.Cleanup(func() { pipelineResultsTemporalClient = orig })
		pipelineResultsTemporalClient = func(string) (client.Client, error) {
			t.Fatal("temporal client must not be requested")
			return nil, nil
		}
		stats, err := namespaceScoreboard(ctx, app, "usera-s-organization")
		require.NoError(t, err)
		assert.Empty(t, stats)
	})

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	createPipelineRecord(t, app, orgID, "cov-namespace")

	orig := pipelineResultsTemporalClient
	t.Cleanup(func() { pipelineResultsTemporalClient = orig })
	pipelineResultsTemporalClient = func(string) (client.Client, error) {
		return nil, errors.New("no temporal")
	}

	_, err = namespaceScoreboard(ctx, app, "usera-s-organization")
	require.ErrorContains(t, err, "create temporal client: no temporal")

	_, err = scoreboardExecutionDetails(app, "usera-s-organization", "wf", "run")
	require.ErrorContains(t, err, "create temporal client: no temporal")

	m := temporalmocks.NewClient(t)
	m.On("DescribeWorkflowExecution", mock.Anything, "wf", "run").
		Return(nil, errors.New("not found")).Once()
	stubScoreboardTemporalClient(t, m)
	_, err = scoreboardExecutionDetails(app, "usera-s-organization", "wf", "run")
	require.ErrorContains(t, err, "describe workflow execution: not found")
}

func TestHandleCancelAggregateScoreboardScheduleDeleteFailure(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()

	handle := temporalmocks.NewScheduleHandle(t)
	handle.On("Delete", mock.Anything).Return(errors.New("temporal unavailable")).Once()
	m := temporalmocks.NewClient(t)
	m.On("ScheduleClient").Return(&fakeScheduleClient{handle: handle})
	m.On("Close").Return().Maybe()

	orig := scheduleTemporalClient
	t.Cleanup(func() { scheduleTemporalClient = orig })
	scheduleTemporalClient = func(string) (client.Client, error) { return m, nil }

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/pipeline/scoreboard/aggregate/schedule/s-1",
		nil,
	)
	req.SetPathValue("schedule_id", "s-1")
	rec := httptest.NewRecorder()
	err := HandleCancelAggregateScoreboardSchedule()(&core.RequestEvent{
		App:   app,
		Event: router.Event{Request: req, Response: rec},
	})
	covAPIRequireAPIError(t, err, http.StatusInternalServerError, "failed to delete schedule")
}
