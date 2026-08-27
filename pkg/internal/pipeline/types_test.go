// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFinallyDefinitionHelpers(t *testing.T) {
	empty := FinallyDefinition{}
	require.True(t, empty.IsZero())
	require.Empty(t, empty.AllSteps())

	definition := FinallyDefinition{
		Always:    []FinallyStepDefinition{{StepSpec: StepSpec{ID: "always"}}},
		OnSuccess: []FinallyStepDefinition{{StepSpec: StepSpec{ID: "success"}}},
		OnFailure: []FinallyStepDefinition{{StepSpec: StepSpec{ID: "failure"}}},
	}
	require.False(t, definition.IsZero())
	require.Equal(t, []string{"always", "success", "failure"}, []string{
		definition.AllSteps()[0].ID,
		definition.AllSteps()[1].ID,
		definition.AllSteps()[2].ID,
	})
}

func TestValidRunType(t *testing.T) {
	require.True(t, ValidRunType(RunTypeManual))
	require.True(t, ValidRunType(RunTypeScheduled))
	require.True(t, ValidRunType(RunTypeCI))
	require.False(t, ValidRunType("unknown"))
}
