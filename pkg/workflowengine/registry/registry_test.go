// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package registry

import (
	"reflect"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/stretchr/testify/require"
)

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

func TestRegistryFactoriesCreateInstances(t *testing.T) {
	t.Parallel()

	for key, factory := range Registry {
		key := key
		factory := factory

		t.Run(key, func(t *testing.T) {
			t.Parallel()

			require.NotNil(t, factory.NewFunc)
			require.NotNil(t, factory.NewFunc())
			require.Contains(t, []TaskKind{TaskActivity, TaskWorkflow}, factory.Kind)
		})
	}
}

func TestRegistryMetadataByTaskKind(t *testing.T) {
	t.Parallel()

	for key, factory := range Registry {
		if factory.Kind == TaskActivity {
			require.NotNilf(t, factory.PayloadType, "expected payload type for activity %s", key)
			require.Contains(
				t,
				[]workflowengine.OutputKind{
					workflowengine.OutputAny,
					workflowengine.OutputString,
					workflowengine.OutputMap,
					workflowengine.OutputArrayOfString,
					workflowengine.OutputArrayOfMap,
					workflowengine.OutputBool,
				},
				factory.OutputKind,
			)
			continue
		}

		require.NotNilf(t, factory.PayloadType, "expected payload type for workflow %s", key)
		if factory.CustomTaskQueue {
			require.NotNilf(
				t,
				factory.PipelinePayloadType,
				"expected pipeline payload type for custom queue task %s",
				key,
			)
		}
	}
}

func TestRegistryExpectedTaskTypeMappings(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		key              string
		expectedPayload  reflect.Type
		expectedPipeline reflect.Type
	}{
		{
			key:             "http-request",
			expectedPayload: reflect.TypeOf(activities.HTTPActivityPayload{}),
		},
		{
			key:              "mobile-automation",
			expectedPayload:  reflect.TypeOf(workflows.MobileAutomationWorkflowPayload{}),
			expectedPipeline: reflect.TypeOf(workflows.MobileAutomationWorkflowPipelinePayload{}),
		},
		{
			key:              "conformance-check",
			expectedPayload:  reflect.TypeOf(workflows.StartCheckWorkflowPayload{}),
			expectedPipeline: reflect.TypeOf(workflows.StartCheckWorkflowPipelinePayload{}),
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.key, func(t *testing.T) {
			t.Parallel()

			task := Registry[tc.key]
			require.Equal(t, tc.expectedPayload, task.PayloadType)
			if tc.expectedPipeline != nil {
				require.Equal(t, tc.expectedPipeline, task.PipelinePayloadType)
			}
		})
	}
}

func TestPipelineInternalRegistryContainsTasks(t *testing.T) {
	t.Parallel()

	require.Contains(t, PipelineInternalRegistry, "scheduled-pipeline-enqueue")
	require.Contains(t, PipelineInternalRegistry, "mobile-runner-semaphore-done")

	task := PipelineInternalRegistry["scheduled-pipeline-enqueue"]
	require.Equal(t, TaskWorkflow, task.Kind)
	require.NotNil(t, task.NewFunc())
}

func TestPipelineInternalRegistryFactoriesCreateInstances(t *testing.T) {
	t.Parallel()

	for key, factory := range PipelineInternalRegistry {
		key := key
		factory := factory

		t.Run(key, func(t *testing.T) {
			t.Parallel()

			require.NotNil(t, factory.NewFunc)
			require.NotNil(t, factory.NewFunc())
			require.Contains(t, []TaskKind{TaskActivity, TaskWorkflow}, factory.Kind)
		})
	}
}

func TestPipelineWorkerDenylist(t *testing.T) {
	t.Parallel()

	_, ok := PipelineWorkerDenylist["mobile-automation"]
	require.True(t, ok)
}
