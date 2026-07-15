// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package engine

import (
	"context"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/catalog"
	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/forkbombeu/credimi/pkg/fcaf/validators"
	"github.com/stretchr/testify/require"
)

func TestPipelinePreconditionSelectsRunFixture(t *testing.T) {
	cat := &catalog.Catalog{
		Tests: map[string]dsl.TestDefinition{
			"fixture-test": {
				ID: "fixture-test", Suite: dsl.Suite{SUT: "wallet_solution", Role: "relying_party"},
				Preconditions: []dsl.PreconditionRef{{Ref: "pipeline.pid"}},
				Evidence:      map[string]dsl.EvidenceBinding{"value": {From: "pipeline.pid.outputs.value"}},
				Assertions:    []dsl.AssertionDefinition{{ID: "value-present", Validator: "evidence.present", Input: "evidence.value"}},
				Verdict:       dsl.VerdictPolicy{PassWhen: "all_assertions_pass"},
			},
		},
		Preconditions: map[string]dsl.PreconditionDefinition{
			"pipeline.pid": {ID: "pipeline.pid", Kind: "pipeline", PipelineID: "real-pid", Fixtures: map[string]string{"beta_capture": "beta-pid"}, Outputs: map[string]dsl.OutputDefinition{"value": {Path: "$.output.value", Decoder: "raw"}}},
		},
	}
	for _, tc := range []struct{ name, fixture, pipeline string }{{"default", "eudiw", "real-pid"}, {"beta", "beta_capture", "beta-pid"}} {
		t.Run(tc.name, func(t *testing.T) {
			engine, err := New(nil)
			require.NoError(t, err)
			report, err := engine.ExecuteCatalog(context.Background(), cat, []string{"fixture-test"}, "", map[string]any{"fixture": tc.fixture}, evidence.Bundle{PipelineOutputs: map[string]any{
				tc.pipeline: map[string]any{"workflowId": tc.name, "output": map[string]any{"value": tc.name}},
			}})
			require.NoError(t, err)
			require.Equal(t, validators.StatusPass, report.Tests[0].Status)
			require.Equal(t, tc.name, report.Tests[0].Evidence[0].Value)
		})
	}
}
