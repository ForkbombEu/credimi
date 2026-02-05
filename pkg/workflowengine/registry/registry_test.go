// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package registry

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
)

func TestRegistryFactoriesAreWellFormed(t *testing.T) {
	for key, factory := range Registry {
		require.NotNil(t, factory.NewFunc, "missing NewFunc for %s", key)
		require.True(
			t,
			factory.Kind == TaskActivity || factory.Kind == TaskWorkflow,
			"invalid kind for %s",
			key,
		)

		if factory.Kind == TaskActivity {
			require.NotNil(t, factory.PayloadType, "missing payload type for %s", key)
			requireOutputKindValid(t, factory.OutputKind, key)
		}
	}
}

func TestPipelineWorkerDenylistEntriesExist(t *testing.T) {
	for key := range PipelineWorkerDenylist {
		_, ok := Registry[key]
		require.True(t, ok, "denylisted key missing from registry: %s", key)
	}
}

func requireOutputKindValid(t *testing.T, kind workflowengine.OutputKind, key string) {
	require.GreaterOrEqual(
		t,
		int(kind),
		int(workflowengine.OutputAny),
		"invalid output kind for %s",
		key,
	)
	require.LessOrEqual(
		t,
		int(kind),
		int(workflowengine.OutputBool),
		"invalid output kind for %s",
		key,
	)
}
