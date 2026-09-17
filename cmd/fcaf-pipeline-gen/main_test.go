// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGenerateCompleteFCAFPipeline(t *testing.T) {
	root := filepath.Join(
		"..",
		"..",
		"config_templates",
		"fcaf",
		"wallet_solution",
		"relying_party",
	)
	output := filepath.Join(t.TempDir(), "complete.yaml")
	require.NoError(t, generate(filepath.Join(root, "scenarios"), output))

	data, err := os.ReadFile(output)
	require.NoError(t, err)
	var definition pipelineDefinition
	require.NoError(t, yaml.Unmarshal(data, &definition))
	require.Len(t, definition.Steps, 646)

	require.Equal(t, "onboard-reference-wallet", definition.Steps[0]["id"])
	validationSteps := make([]map[string]any, 0, 1)
	for index, step := range definition.Steps {
		if step["use"] == validationTask {
			validationSteps = append(validationSteps, step)
			continue
		}
		if step["id"] == "onboard-reference-wallet" {
			continue
		}
		require.Equal(t, true, step["continue_on_error"], "step %d", index)
	}
	require.Len(t, validationSteps, 1)
	with, ok := validationSteps[0]["with"].(map[string]any)
	require.True(t, ok)
	require.Len(t, stringSlice(with["test_ids"]), 579)
	require.Contains(t, stringSlice(with["test_ids"]), "WS_RP_SM_RpIntegrity__006")
	require.Contains(t, stringSlice(with["test_ids"]), "WS_RP_SM_RpIntegrity__013")
	for _, restricted := range []string{
		"WS_RP_SM_SessionEncryption__002",
		"WS_RP_SM_SessionEncryption__003",
		"WS_RP_SM_SessionEncryption__007",
		"WS_RP_SM_SessionEncryption__008",
		"WS_RP_SM_SessionEncryption__009",
	} {
		require.Contains(
			t,
			stringSlice(with["test_ids"]),
			restricted,
			"each response-encryption case needs its own verifier metadata scenario",
		)
	}
	require.Len(t, with["pipeline_outputs"], 181)

	committed, err := os.ReadFile(filepath.Join(
		root,
		"pipelines",
		"fcaf-wallet-solution-relying-party-complete-validation.yaml",
	))
	require.NoError(t, err)
	require.Equal(t, committed, data, "generated aggregate pipeline is stale")
}

func TestGenerateDemoFCAFPipeline(t *testing.T) {
	root := filepath.Join(
		"..",
		"..",
		"config_templates",
		"fcaf",
		"wallet_solution",
		"relying_party",
	)
	output := filepath.Join(t.TempDir(), "demo.yaml")
	require.NoError(t, generateDemo(filepath.Join(root, "scenarios"), output))

	data, err := os.ReadFile(output)
	require.NoError(t, err)
	var definition pipelineDefinition
	require.NoError(t, yaml.Unmarshal(data, &definition))
	require.Len(t, definition.Steps, 7)
	require.Equal(t, "onboard-reference-wallet", definition.Steps[0]["id"])

	validation := definition.Steps[len(definition.Steps)-1]
	require.Equal(t, validationTask, validation["use"])
	with, ok := validation["with"].(map[string]any)
	require.True(t, ok)
	selectedTestIDs := stringSlice(with["test_ids"])
	require.Len(t, selectedTestIDs, 46)
	require.ElementsMatch(t, demoTestIDs, selectedTestIDs)
	require.Len(t, with["pipeline_outputs"], 3)

	committed, err := os.ReadFile(filepath.Join(
		root,
		"pipelines",
		"fcaf-wallet-solution-relying-party-demo-validation.yaml",
	))
	require.NoError(t, err)
	require.Equal(t, committed, data, "generated demo pipeline is stale")
}

func TestGenerateHappyFlowFCAFPipeline(t *testing.T) {
	root := filepath.Join(
		"..",
		"..",
		"config_templates",
		"fcaf",
		"wallet_solution",
		"relying_party",
	)
	output := filepath.Join(t.TempDir(), "happy-flow.yaml")
	require.NoError(t, generateHappyFlow(filepath.Join(root, "scenarios"), output))

	data, err := os.ReadFile(output)
	require.NoError(t, err)
	var definition pipelineDefinition
	require.NoError(t, yaml.Unmarshal(data, &definition))
	require.Len(t, definition.Steps, 118)
	require.Equal(t, "onboard-reference-wallet", definition.Steps[0]["id"])

	validationSteps := make([]map[string]any, 0, 1)
	for index, step := range definition.Steps {
		if step["use"] == validationTask {
			validationSteps = append(validationSteps, step)
			continue
		}
		if step["id"] == "onboard-reference-wallet" {
			continue
		}
		require.Equal(t, true, step["continue_on_error"], "step %d", index)
	}
	require.Len(t, validationSteps, 1)
	with, ok := validationSteps[0]["with"].(map[string]any)
	require.True(t, ok)
	require.NotContains(
		t,
		stringSlice(with["test_ids"]),
		"WS_RP_IA_MainInteraction__015",
		"happy flow must omit tests whose exact evidence source is not selected",
	)
	require.Len(t, stringSlice(with["test_ids"]), 383)
	require.NotContains(
		t,
		stringSlice(with["test_ids"]),
		"WS_RP_SM_RpIntegrity__006",
		"happy flow must omit tests whose exact evidence source is not selected",
	)
	require.NotContains(
		t,
		stringSlice(with["test_ids"]),
		"WS_RP_SM_RpIntegrity__013",
		"happy flow must omit tests whose exact evidence source is not selected",
	)
	require.NotContains(
		t,
		stringSlice(with["test_ids"]),
		"WS_RP_SM_RpIntegrity__016",
		"happy flow must omit tests without exact feasible evidence",
	)
	require.NotContains(
		t,
		stringSlice(with["test_ids"]),
		"WS_RP_SM_RpIntegrity__018",
		"happy flow must omit tests without exact feasible evidence",
	)
	require.NotContains(
		t,
		stringSlice(with["test_ids"]),
		"WS_RP_SM_RpIntegrity__020",
		"happy flow must omit tests without exact feasible evidence",
	)
	require.Len(t, with["pipeline_outputs"], 34)

	committed, err := os.ReadFile(filepath.Join(
		root,
		"pipelines",
		"fcaf-wallet-solution-relying-party-happy-flow-validation.yaml",
	))
	require.NoError(t, err)
	require.Equal(t, committed, data, "generated happy flow pipeline is stale")
}
