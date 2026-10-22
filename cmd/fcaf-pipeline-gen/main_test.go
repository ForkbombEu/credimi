// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
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
	require.Len(t, definition.Steps, 1452)
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
	require.Len(t, stringSlice(with["test_ids"]), 617)
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
	require.Len(t, with["pipeline_outputs"], 215)

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
	require.Len(t, definition.Steps, 141)
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
	require.Len(t, stringSlice(with["test_ids"]), 289)
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
	require.Len(t, with["pipeline_outputs"], 32)

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
// every wallet action runs after its own scenario's reset, a presentation that
// can reach Share is preceded by an unspent issuance of the format it requests
// from the same scenario, and a presentation the wallet refuses still finds a
// PID of that format, so the refusal is not just an empty wallet. A step's
// `actions` run in order, so each entry counts as a wallet step of its own.
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
	pidSession := regexp.MustCompile(`^(.+)-issue-pid(?:-[0-9]+)?-session$`)
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
		previousActionResets := false
		resets := 0
		shares := 0
		injected := 0
		for index, step := range definition.Steps {
			id, _ := step["id"].(string)
			if use, _ := step["use"].(string); use != mobileAutomationTask {
				earlier[id] = step
				continue
			}
			for _, action := range walletStepActions(t, step) {
				source := deeplinkSourceStep(action, earlier)
				with, _ := action["with"].(map[string]any)
				actionID, _ := with["action_id"].(string)
				if actionID == resetActionID {
					require.Falsef(
						t,
						previousActionResets,
						"%s: %q resets a wallet that nothing used since the last reset",
						pipeline,
						id,
					)
					previousActionResets = true
					scenario = afterPrelude
					if id != "onboard-reference-wallet" {
						match := scenarioOf.FindStringSubmatch(id)
						require.NotNilf(
							t,
							match,
							"%s: step %q has no scenario prefix",
							pipeline,
							id,
						)
						scenario = match[1]
					}
					available = map[credentialFormat]int{}
					resets++
					continue
				}
				previousActionResets = false
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
				if issuesCredential(action) {
					available[issuedCredentialFormat(source)]++
					if actionID != injectedIssuanceActionID {
						continue
					}
					// An injected issuance accepts the offer of an issuer
					// session created for its presentation, and runs in that
					// presentation's step or, for a presentation driving
					// another app, in its own step right before it.
					injected++
					sessionID, _ := source["id"].(string)
					match := pidSession.FindStringSubmatch(sessionID)
					require.NotNilf(
						t,
						match,
						"%s: %q issues a PID from %q, not an injected issuer session",
						pipeline,
						id,
						sessionID,
					)
					if presentation := match[1]; presentation != id {
						own := injectedFor.FindStringSubmatch(id)
						require.Truef(
							t,
							own != nil && own[1] == presentation &&
								slices.ContainsFunc(
									definition.Steps[index+1:],
									func(later map[string]any) bool { return later["id"] == presentation },
								),
							"%s: injected issuance %q is not followed by its presentation %q",
							pipeline,
							id,
							presentation,
						)
					}
					continue
				}
				consumed, err := credentialInstancesConsumed(action, actions)
				require.NoError(t, err)
				shares += consumed
				// A scenario-authored issuance may deliberately stay unspent, for
				// example to prove a held credential does not match the request,
				// but every Share must find an unspent instance of the requested
				// format.
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
				// A multiple: true query returns, and so spends, every instance
				// of its format the wallet holds.
				if consumed > 0 {
					for _, format := range formatsSpentEntirely(source) {
						available[format] = 0
					}
				}
				if consumed == 0 && opensPresentationRequest(action, source) {
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
			earlier[id] = step
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

// walletStepActions returns a wallet step's actions, in run order, each as a
// single-action step carrying the step's id: an `actions` entry gets the
// step's parameters, overridden by its own.
func walletStepActions(t *testing.T, step map[string]any) []map[string]any {
	t.Helper()
	with, _ := step["with"].(map[string]any)
	entries, listed := with["actions"].([]any)
	if !listed {
		return []map[string]any{step}
	}
	require.NotContains(t, with, "action_id", "step %v", step["id"])
	require.NotContains(t, with, "action_code", "step %v", step["id"])
	shared, _ := with["parameters"].(map[string]any)
	single := make([]map[string]any, 0, len(entries))
	for _, raw := range entries {
		entry, _ := raw.(map[string]any)
		action := map[string]any{}
		if id, ok := entry["id"]; ok {
			action["action_id"] = id
		}
		if code, ok := entry["code"]; ok {
			action["action_code"] = code
		}
		parameters := maps.Clone(shared)
		if parameters == nil {
			parameters = map[string]any{}
		}
		own, _ := entry["parameters"].(map[string]any)
		maps.Copy(parameters, own)
		action["parameters"] = parameters
		single = append(single, map[string]any{
			"id":   step["id"],
			"use":  step["use"],
			"with": action,
		})
	}
	return single
}
