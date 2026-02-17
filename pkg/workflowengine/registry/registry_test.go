// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package registry

import "testing"

import "github.com/stretchr/testify/require"

func TestRegistryContainsCoreTasks(t *testing.T) {
	t.Parallel()

	require.Contains(t, Registry, "http-request")
	require.Contains(t, Registry, "mobile-automation")
	require.Contains(t, Registry, "conformance-check")

	httpTask := Registry["http-request"]
	require.Equal(t, TaskActivity, httpTask.Kind)
	require.NotNil(t, httpTask.NewFunc())
	require.NotNil(t, httpTask.PayloadType)

	mobileTask := Registry["mobile-automation"]
	require.Equal(t, TaskWorkflow, mobileTask.Kind)
	require.NotNil(t, mobileTask.NewFunc())
	require.NotNil(t, mobileTask.PayloadType)
	require.True(t, mobileTask.CustomTaskQueue)
	require.NotNil(t, mobileTask.PipelinePayloadType)
}

func TestPipelineInternalRegistryContainsTasks(t *testing.T) {
	t.Parallel()

	require.Contains(t, PipelineInternalRegistry, "scheduled-pipeline-enqueue")
	require.Contains(t, PipelineInternalRegistry, "mobile-runner-semaphore-done")

	task := PipelineInternalRegistry["scheduled-pipeline-enqueue"]
	require.Equal(t, TaskWorkflow, task.Kind)
	require.NotNil(t, task.NewFunc())
}

func TestPipelineWorkerDenylist(t *testing.T) {
	t.Parallel()

	_, ok := PipelineWorkerDenylist["mobile-automation"]
	require.True(t, ok)
}
