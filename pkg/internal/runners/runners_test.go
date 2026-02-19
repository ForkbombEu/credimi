// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package runners

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const testDataDir = "../../../test_pb_data"

func TestParsePipelineRunnerInfo(t *testing.T) {
	t.Run("empty yaml returns zero value", func(t *testing.T) {
		got, err := ParsePipelineRunnerInfo("   ")
		require.NoError(t, err)
		require.False(t, got.NeedsGlobalRunner)
		require.Empty(t, got.RunnerIDs)
	})

	t.Run("invalid yaml returns error", func(t *testing.T) {
		_, err := ParsePipelineRunnerInfo("[")
		require.Error(t, err)
	})

	t.Run("collects and deduplicates runner ids from steps and branches", func(t *testing.T) {
		yamlStr := `
name: test
steps:
  - id: step-1
    use: mobile-automation
    with:
      runner_id: runner-b
  - id: step-2
    use: mobile-automation
    with:
      payload:
        runner_id: runner-a
  - id: step-3
    use: echo
    with:
      message: ok
    on_error:
      - id: err-step
        use: mobile-automation
        with:
          runner_id: runner-c
    on_success:
      - id: success-step
        use: mobile-automation
        with:
          payload:
            runner_id: runner-a
  - id: step-4
    use: mobile-automation
    with:
      action_id: missing-runner-id
`

		got, err := ParsePipelineRunnerInfo(yamlStr)
		require.NoError(t, err)
		require.True(t, got.NeedsGlobalRunner)
		require.Equal(t, []string{"runner-a", "runner-b", "runner-c"}, got.RunnerIDs)
	})
}

func TestRunnerIDsWithGlobal(t *testing.T) {
	t.Run("adds global runner if needed and missing", func(t *testing.T) {
		info := PipelineRunnerInfo{
			RunnerIDs:         []string{"runner-b"},
			NeedsGlobalRunner: true,
		}
		got := RunnerIDsWithGlobal(info, " runner-a ")
		require.Equal(t, []string{"runner-a", "runner-b"}, got)
	})

	t.Run("does not duplicate global runner", func(t *testing.T) {
		info := PipelineRunnerInfo{
			RunnerIDs:         []string{"runner-a", "runner-b"},
			NeedsGlobalRunner: true,
		}
		got := RunnerIDsWithGlobal(info, "runner-a")
		require.Equal(t, []string{"runner-a", "runner-b"}, got)
	})

	t.Run("ignores global runner when not needed", func(t *testing.T) {
		info := PipelineRunnerInfo{
			RunnerIDs:         []string{"runner-a"},
			NeedsGlobalRunner: false,
		}
		got := RunnerIDsWithGlobal(info, "runner-b")
		require.Equal(t, []string{"runner-a"}, got)
	})
}

func TestGlobalRunnerIDFromConfig(t *testing.T) {
	require.Equal(t, "", GlobalRunnerIDFromConfig(nil))
	require.Equal(t, "", GlobalRunnerIDFromConfig(map[string]any{"global_runner_id": 12}))
	require.Equal(
		t,
		"runner-a",
		GlobalRunnerIDFromConfig(map[string]any{"global_runner_id": " runner-a "}),
	)
}

func TestResolveRunnerRecord(t *testing.T) {
	t.Run("empty runner id returns nil", func(t *testing.T) {
		got := ResolveRunnerRecord(nil, " ", nil)
		require.Nil(t, got)
	})

	t.Run("returns cached record", func(t *testing.T) {
		cache := map[string]map[string]any{
			"runner-a": {"id": "cached-id"},
		}
		got := ResolveRunnerRecord(nil, "runner-a", cache)
		require.Equal(t, map[string]any{"id": "cached-id"}, got)
	})

	t.Run("not found runner is cached as nil", func(t *testing.T) {
		app, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		defer app.Cleanup()

		cache := map[string]map[string]any{}
		runnerID := "missing-org/missing-runner"

		got := ResolveRunnerRecord(app, runnerID, cache)
		require.Nil(t, got)
		_, ok := cache[runnerID]
		require.True(t, ok)
		require.Nil(t, cache[runnerID])
	})

	t.Run("resolves known runner from fixtures", func(t *testing.T) {
		app, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		defer app.Cleanup()

		org, err := app.FindFirstRecordByFilter("organizations", "1=1")
		require.NoError(t, err)
		orgCanon := org.GetString("canonified_name")
		require.NotEmpty(t, orgCanon)

		runnersCollection, err := app.FindCollectionByNameOrId("mobile_runners")
		require.NoError(t, err)
		newRunner := core.NewRecord(runnersCollection)
		newRunner.Set("owner", org.Id)
		newRunner.Set("name", "test-runner")
		newRunner.Set("canonified_name", "test-runner")
		newRunner.Set("ip", "127.0.0.1")
		require.NoError(t, app.Save(newRunner))

		cache := map[string]map[string]any{}
		runnerID := orgCanon + "/test-runner"
		got := ResolveRunnerRecord(app, runnerID, cache)

		require.NotNil(t, got)
		require.NotEmpty(t, got[core.FieldNameId])
		require.Equal(t, got, cache[runnerID])
	})
}

func TestResolveRunnerRecords(t *testing.T) {
	t.Run("empty input returns empty slice", func(t *testing.T) {
		got := ResolveRunnerRecords(nil, nil, nil)
		require.Empty(t, got)
	})

	t.Run("returns only resolvable runner records", func(t *testing.T) {
		app, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		defer app.Cleanup()

		org, err := app.FindFirstRecordByFilter("organizations", "1=1")
		require.NoError(t, err)
		orgCanon := org.GetString("canonified_name")
		require.NotEmpty(t, orgCanon)

		runnersCollection, err := app.FindCollectionByNameOrId("mobile_runners")
		require.NoError(t, err)
		newRunner := core.NewRecord(runnersCollection)
		newRunner.Set("owner", org.Id)
		newRunner.Set("name", "queue-runner")
		newRunner.Set("canonified_name", "queue-runner")
		newRunner.Set("ip", "127.0.0.1")
		require.NoError(t, app.Save(newRunner))

		cache := map[string]map[string]any{}
		resolvableRunnerID := orgCanon + "/queue-runner"
		got := ResolveRunnerRecords(
			app,
			[]string{
				resolvableRunnerID,
				"missing-org/missing-runner",
			},
			cache,
		)

		require.Len(t, got, 1)
		require.NotEmpty(t, got[0][core.FieldNameId])
	})
}
