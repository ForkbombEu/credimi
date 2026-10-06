// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"errors"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/catalog"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	temporalmocks "go.temporal.io/sdk/mocks"
)

type covWEFailingHistoryIterator struct{}

func (covWEFailingHistoryIterator) HasNext() bool { return true }

func (covWEFailingHistoryIterator) Next() (*historypb.HistoryEvent, error) {
	return nil, errors.New("history unavailable")
}

func covWEMockFCAFHistory(t *testing.T, iterator any) {
	t.Helper()
	c := &temporalmocks.Client{}
	c.On(
		"GetWorkflowHistory",
		mock.Anything,
		fcafTestWorkflowID,
		fcafTestRunID,
		false,
		enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	).Return(iterator)
	temporalclient.SetClientForTests(fcafTestNamespace, c)
}

func TestCovWEFCAFValidationActivityRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		payload any
	}{
		{name: "payload is not an object", payload: "not-an-object"},
		{
			name: "blank test ids only",
			payload: FCAFValidationActivityInput{
				TestIDs:  []string{" ", ""},
				Pipeline: map[string]any{"s": map[string]any{}},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			act := NewFCAFValidationActivity(nil, fcafTestOutputKind)
			_, err := act.Execute(
				context.Background(),
				workflowengine.ActivityInput{Payload: tc.payload},
			)
			requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, false)
		})
	}
}

func TestCovWEFCAFValidationActivityCatalogLoadFailure(t *testing.T) {
	act := NewFCAFValidationActivity(nil, fcafTestOutputKind)
	var loadedRoot string
	act.catalogLoader = func(root string) (*catalog.Catalog, error) {
		loadedRoot = root
		return nil, errors.New("catalog missing")
	}

	_, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: FCAFValidationActivityInput{
			TestID:      fcafTestTestID,
			CatalogRoot: "/custom/catalog",
			Pipeline:    map[string]any{"s": map[string]any{"value": "static"}},
		},
	})

	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, false)
	assert.Contains(t, err.Error(), "catalog missing")
	assert.Equal(t, "/custom/catalog", loadedRoot)
}

func TestCovWEFCAFValidationActivityUnknownTestFails(t *testing.T) {
	setFCAFTestEnv(t)
	_, err := executeFCAFValidation(t, newPipelineResultsTestApp(t), workflowengine.ActivityInput{
		Payload: FCAFValidationActivityInput{
			TestIDs:  []string{"NOT_A_REAL_FCAF_TEST"},
			Pipeline: map[string]any{fcafTestSource: map[string]any{"output": map[string]any{}}},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), errorcodes.Codes[errorcodes.PipelineExecutionError].Code)
	assert.Contains(t, err.Error(), "execute fcaf engine")
}

func TestCovWEFCAFValidationActivityHistoryFailures(t *testing.T) {
	tests := []struct {
		name     string
		iterator func(t *testing.T) any
		wantCode string
		wantText string
	}{
		{
			name:     "history cannot be loaded",
			iterator: func(*testing.T) any { return covWEFailingHistoryIterator{} },
			wantCode: errorcodes.Codes[errorcodes.PipelineExecutionError].Code,
			wantText: "load pipeline history",
		},
		{
			name:     "expression references an unknown step",
			iterator: func(t *testing.T) any { return &fcafHistoryIterator{events: fcafHistory(t, "x")} },
			wantCode: errorcodes.Codes[errorcodes.PipelineInputError].Code,
			wantText: "resolve pipeline_outputs",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setFCAFTestEnv(t)
			covWEMockFCAFHistory(t, tc.iterator(t))

			_, err := executeFCAFValidation(
				t,
				newPipelineResultsTestApp(t),
				workflowengine.ActivityInput{
					Payload: FCAFValidationActivityInput{
						TestIDs: []string{fcafTestTestID},
						Pipeline: map[string]any{
							fcafTestSource: map[string]any{
								"v": "${{ missing.outputs.body | nope }}",
							},
						},
					},
				},
			)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantCode)
			assert.Contains(t, err.Error(), tc.wantText)
		})
	}
}

func TestCovWEFCAFEvidencePathHelpers(t *testing.T) {
	resolved := map[string]any{
		"a": map[string]any{"list": []any{"x", map[string]any{"leaf": 1}}},
	}
	tests := []struct {
		name     string
		path     []any
		want     any
		wantPath string
	}{
		{
			name:     "nested map and list",
			path:     []any{"a", "list", 1, "leaf"},
			want:     1,
			wantPath: "a.list[1].leaf",
		},
		{name: "index on non list", path: []any{"a", 0}, want: nil, wantPath: "a[0]"},
		{name: "index out of range", path: []any{"a", "list", 5}, want: nil, wantPath: "a.list[5]"},
		{
			name:     "key on non map",
			path:     []any{"a", "list", 0, "k"},
			want:     nil,
			wantPath: "a.list[0].k",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, valueAtPath(resolved, tc.path))
			assert.Equal(t, tc.wantPath, formatLeafPath(tc.path))
		})
	}

	assert.Equal(t, "step", refStepID("step.outputs.value"))
	assert.Equal(t, "step", refStepID("step[0].outputs"))
	assert.Empty(t, refStepID("step.outputs | "))
}
