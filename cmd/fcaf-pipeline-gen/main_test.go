// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	require.Len(t, definition.Steps, 1745)
	require.NotContains(
		t,
		string(data),
		"${fixture.issuer_url}",
		"PID issuance must target the scenario issuer, not pipeline.DefaultIssuerURL",
	)

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
	require.Len(t, stringSlice(with["test_ids"]), 615)
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
	require.Len(t, with["pipeline_outputs"], 220)

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
	require.Len(t, definition.Steps, 195)
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
	require.Len(t, stringSlice(with["test_ids"]), 301)
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
	require.Len(t, with["pipeline_outputs"], 36)

	committed, err := os.ReadFile(filepath.Join(
		root,
		"pipelines",
		"fcaf-wallet-solution-relying-party-happy-flow-validation.yaml",
	))
	require.NoError(t, err)
	require.Equal(t, committed, data, "generated happy flow pipeline is stale")
}

// TestAggregateHoldsACredentialForEveryPresentation guards the reference
// wallet's single-use credential instances and the per-scenario wallet reset:
// every wallet step runs after its own scenario's reset, a presentation that can
// reach Share is preceded by an unspent issuance of the format it requests from
// the same scenario, and a presentation the wallet refuses still finds a PID of
// that format, so the refusal is not just an empty wallet.
func TestAggregateHoldsACredentialForEveryPresentation(t *testing.T) {
	root := filepath.Join(
		"..",
		"..",
		"config_templates",
		"fcaf",
		"wallet_solution",
		"relying_party",
	)
	actions, err := loadWalletActions(filepath.Join(root, "..", "..", "imports"))
	require.NoError(t, err)
	injectedFor := regexp.MustCompile(`^(.+)-issue-pid(?:-[0-9]+)?$`)
	scenarioOf := regexp.MustCompile(`^(.+?-[0-9a-f]{8})-`)

	for _, pipeline := range []string{
		"fcaf-wallet-solution-relying-party-complete-validation.yaml",
		"fcaf-wallet-solution-relying-party-happy-flow-validation.yaml",
		"fcaf-wallet-solution-relying-party-demo-validation.yaml",
	} {
		data, err := os.ReadFile(filepath.Join(root, "pipelines", pipeline))
		require.NoError(t, err)
		var definition pipelineDefinition
		require.NoError(t, yaml.Unmarshal(data, &definition))

		available := map[credentialFormat]int{}
		earlier := map[string]map[string]any{}
		// The first step resets the wallet for the first scenario that drives
		// it; afterPrelude marks that this scenario is not known yet.
		const afterPrelude = "*"
		scenario := ""
		previousWalletStepResets := false
		resets := 0
		shares := 0
		injected := 0
		for index, step := range definition.Steps {
			id, _ := step["id"].(string)
			source := deeplinkSourceStep(step, earlier)
			earlier[id] = step
			if use, _ := step["use"].(string); use != mobileAutomationTask {
				continue
			}
			with, _ := step["with"].(map[string]any)
			if action, _ := with["action_id"].(string); action == resetActionID {
				require.Falsef(
					t,
					previousWalletStepResets,
					"%s: %q resets a wallet that nothing used since the last reset",
					pipeline,
					id,
				)
				previousWalletStepResets = true
				scenario = strings.TrimSuffix(id, "-reset-wallet")
				if id == "onboard-reference-wallet" {
					scenario = afterPrelude
				}
				available = map[credentialFormat]int{}
				resets++
				continue
			}
			previousWalletStepResets = false
			if scenario == afterPrelude {
				match := scenarioOf.FindStringSubmatch(id)
				require.NotNilf(t, match, "%s: step %q has no scenario prefix", pipeline, id)
				scenario = match[1]
			}
			require.Truef(
				t,
				scenario != "" && strings.HasPrefix(id, scenario+"-"),
				"%s: wallet step %q runs without its scenario's wallet reset",
				pipeline,
				id,
			)
			if issuesCredential(step) {
				available[issuedCredentialFormat(source)]++
				if match := injectedFor.FindStringSubmatch(id); match != nil {
					injected++
					// An injected issuance exists only for the presentation it
					// is named after, which must follow in the same scenario.
					followed := false
					for _, later := range definition.Steps[index+1:] {
						laterID, _ := later["id"].(string)
						if laterID == match[1] {
							followed = true
							break
						}
					}
					require.Truef(
						t,
						followed,
						"%s: injected issuance %q is never followed by its presentation",
						pipeline,
						id,
					)
				}
				continue
			}
			consumed, err := credentialInstancesConsumed(step, actions)
			require.NoError(t, err)
			shares += consumed
			// A scenario-authored issuance may deliberately stay unspent, for
			// example to prove a held credential does not match the request, but
			// every Share must find an unspent instance of the requested format.
			for _, format := range presentedCredentialFormats(source, consumed) {
				require.GreaterOrEqualf(
					t,
					available[format],
					consumed,
					"%s: presentation %q shares %d %s instances with %d unspent",
					pipeline,
					id,
					consumed,
					format,
					available[format],
				)
				available[format] -= consumed
			}
			if consumed == 0 && opensPresentationRequest(step, source) {
				for _, format := range presentedCredentialFormats(source, 1) {
					require.Positivef(
						t,
						available[format],
						"%s: presentation %q runs on a wallet holding no %s PID",
						pipeline,
						id,
						format,
					)
				}
			}
		}
		require.Positivef(t, resets, "%s: no scenario resets the wallet", pipeline)
		require.Positivef(t, shares, "%s: no presentation shares a credential", pipeline)
		t.Logf(
			"%s: %d resets, %d shares, %d injected issuances",
			pipeline,
			resets,
			shares,
			injected,
		)
	}
}
