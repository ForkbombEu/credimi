// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/aggregate"
	"github.com/forkbombeu/credimi/pkg/fcaf/walletroles"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// testFCAFRoot is the repository's FCAF templates root.
var testFCAFRoot = filepath.Join("..", "..", "config_templates", "fcaf")

func TestGenerateCompleteFCAFPipeline(t *testing.T) {
	contract, err := walletroles.Load(testFCAFRoot)
	require.NoError(t, err)
	// The action_id prefix of the reference wallet's roles.
	prefix := contract.ReferencePrefix
	output := filepath.Join(t.TempDir(), "complete.yaml")
	require.NoError(t, generate(testFCAFRoot, relyingPartySuite, output))

	data, err := os.ReadFile(output)
	require.NoError(t, err)
	var definition walletroles.Scenario
	require.NoError(t, yaml.Unmarshal(data, &definition))
	require.NotContains(
		t,
		string(data),
		"${fixture.issuer_url}",
		"PID issuance must target the scenario issuer, not pipeline.DefaultIssuerURL",
	)

	require.Equal(t, "onboard-reference-wallet", definition.Steps[0]["id"])
	validationSteps := make([]map[string]any, 0, 1)
	for index, step := range definition.Steps {
		if step["id"] == aggregate.ValidationStepID {
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
	require.Len(t, walletroles.StringList(with["test_ids"]), 617)
	require.Contains(t, walletroles.StringList(with["test_ids"]), "WS_RP_SM_RpIntegrity__006")
	require.Contains(t, walletroles.StringList(with["test_ids"]), "WS_RP_SM_RpIntegrity__013")
	for _, restricted := range []string{
		"WS_RP_SM_SessionEncryption__002",
		"WS_RP_SM_SessionEncryption__003",
		"WS_RP_SM_SessionEncryption__007",
		"WS_RP_SM_SessionEncryption__008",
		"WS_RP_SM_SessionEncryption__009",
	} {
		require.Contains(
			t,
			walletroles.StringList(with["test_ids"]),
			restricted,
			"each response-encryption case needs its own verifier metadata scenario",
		)
	}
	require.Len(t, with["pipeline_outputs"], 216)

	stepsByID := make(map[string]map[string]any, len(definition.Steps))
	for _, step := range definition.Steps {
		id, _ := step["id"].(string)
		stepsByID[id] = step
	}
	observedRequest := stepsByID["client-id-prefix-controls-f49aa8ab-observe-signed-redirect-uri"]
	observedRequestWith, _ := observedRequest["with"].(map[string]any)
	require.Contains(
		t,
		observedRequestWith["actions"],
		map[string]any{
			"id": prefix + "fcaf-observe-request",
			"parameters": map[string]any{
				"deeplink": "${{ client-id-prefix-controls-f49aa8ab-create-signed-redirect-uri.outputs.body.deeplink }}",
			},
		},
	)
	require.Contains(
		t,
		observedRequestWith["actions"],
		map[string]any{
			"id": prefix + "fcaf-issue-pid",
			"parameters": map[string]any{
				"deeplink": "${{ client-id-prefix-controls-f49aa8ab-observe-signed-redirect-uri-issue-pid-session.outputs.body.deeplink }}",
			},
		},
	)
	reusedCredential := stepsByID["client-id-prefix-controls-f49aa8ab-reject-origin-outside-dc-api"]
	reusedCredentialWith, _ := reusedCredential["with"].(map[string]any)
	require.Equal(
		t,
		prefix+"fcaf-expect-request-rejected",
		reusedCredentialWith["action_id"],
	)
	require.NotContains(t, reusedCredentialWith, "actions")
	stringVariants := []string{"null", "true", "false", "zero", "number", "string", "object"}
	arrayVariants := []string{"null", "true", "false", "zero", "number", "array", "object"}
	for _, scenario := range []struct {
		prefix   string
		variants []string
	}{
		{prefix: "dcql-claim-sets-non-array-1ebdf3bd-", variants: stringVariants},
		{prefix: "dcql-claims-non-array-ed748dc5-", variants: stringVariants},
		{prefix: "dcql-trusted-authorities-type-non-st-4e26a188-", variants: arrayVariants},
		{prefix: "dcql-trusted-authorities-values-item-000a74ab-", variants: arrayVariants},
		{prefix: "dcql-trusted-authorities-values-non-2ccb8dca-", variants: stringVariants},
	} {
		for _, variant := range scenario.variants {
			exercise := stepsByID[scenario.prefix+"exercise-"+variant]
			exerciseWith, _ := exercise["with"].(map[string]any)
			actions, ok := exerciseWith["actions"].([]any)
			require.True(
				t,
				ok,
				"%sexercise-%s must compose wallet actions",
				scenario.prefix,
				variant,
			)
			require.NotEmpty(
				t,
				actions,
				"%sexercise-%s must run the strict rejection action",
				scenario.prefix,
				variant,
			)
			action, ok := actions[len(actions)-1].(map[string]any)
			require.True(t, ok)
			require.Equal(t, prefix+"fcaf-expect-request-rejected", action["id"])
			parameters, _ := action["parameters"].(map[string]any)
			require.Equal(
				t,
				"${{ "+scenario.prefix+"create-"+variant+".outputs.body.deeplink }}",
				parameters["deeplink"],
			)
			require.NotContains(
				t,
				stepsByID[scenario.prefix+"fetch-"+variant],
				"linked_to",
				"%sfetch-%s must run after a failed rejection action",
				scenario.prefix,
				variant,
			)
		}
	}
	holderBindingPrefix := "dcql-holder-binding-type-boundary-3ff3e611-"
	for _, stepID := range []string{
		"get-valid-true",
		"get-valid-false",
		"fetch-number-zero",
		"fetch-number-one",
		"fetch-number-random",
		"fetch-string-true",
		"fetch-string-false",
		"fetch-string-uppercase-true",
		"fetch-string-uppercase-false",
		"fetch-string-yes",
		"fetch-string-no",
		"fetch-string-y",
		"fetch-string-n",
		"fetch-string-zero",
		"fetch-string-one",
		"fetch-string-random",
	} {
		require.NotContains(
			t,
			stepsByID[holderBindingPrefix+stepID],
			"linked_to",
			"%s must run after a failed wallet action",
			stepID,
		)
	}
	require.Equal(
		t,
		"dcql-credentials-match-cc301c32-create-pid-sdjwt-offer",
		stepsByID["dcql-credentials-match-cc301c32-obtain-pid-sdjwt"]["linked_to"],
	)
	require.Equal(
		t,
		"callback-redirect-298d09a0-exercise-wallet-issue-pid-session",
		stepsByID["callback-redirect-298d09a0-exercise-wallet"]["linked_to"],
	)
	require.Equal(
		t,
		"callback-redirect-298d09a0-create-callback-session",
		stepsByID["callback-redirect-298d09a0-exercise-wallet-issue-pid-session"]["linked_to"],
	)
	presentation := stepsByID["dcql-credentials-match-cc301c32-create-presentation"]
	require.NotNil(t, presentation)
	presentationOptions, _ := presentation["activity_options"].(map[string]any)
	require.NotContains(t, presentationOptions, "retry_policy")
	walletOptions, _ := stepsByID["dcql-credentials-match-cc301c32-present-with-reference-implementation"]["activity_options"].(map[string]any)
	walletPolicy, _ := walletOptions["retry_policy"].(map[string]any)
	require.Equal(
		t,
		3,
		walletPolicy["maximum_attempts"],
		"the chain keeps the producer's retry policy",
	)
}

func TestGenerateAttestationProviderFCAFPipeline(t *testing.T) {
	root := filepath.Join("..", "..", "config_templates", "fcaf")
	output := filepath.Join(t.TempDir(), "complete.yaml")
	require.NoError(t, generate(root, attestationProviderSuite, output))

	data, err := os.ReadFile(output)
	require.NoError(t, err)
	var definition walletroles.Scenario
	require.NoError(t, yaml.Unmarshal(data, &definition))
	require.Equal(t, "FCAF wallet attestation-provider complete validation", definition.Name)
	require.Equal(t, "onboard-reference-wallet", definition.Steps[0]["id"])

	validation := definition.Steps[len(definition.Steps)-1]
	require.Equal(t, aggregate.ValidationStepID, validation["id"])
	with, ok := validation["with"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, attestationProviderSuite, with["suite"])
	testIDs := walletroles.StringList(with["test_ids"])
	require.Len(t, testIDs, 59)
	for _, id := range testIDs {
		require.True(t, strings.HasPrefix(id, "WS_AP_"), id)
	}
	for _, step := range definition.Steps[1 : len(definition.Steps)-1] {
		id, _ := step["id"].(string)
		require.False(
			t,
			strings.HasPrefix(id, "fcaf-wallet-solution-attestation-provider-"),
			"scenario step ids drop the suite file-name prefix: %s",
			id,
		)
	}

	committed, err := os.ReadFile(suitePipelinePath(root, attestationProviderSuite, "complete"))
	require.NoError(t, err)
	require.Equal(t, committed, data, "generated attestation-provider pipeline is stale")
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
	require.NoError(t, generateDemo(testFCAFRoot, output))

	data, err := os.ReadFile(output)
	require.NoError(t, err)
	var definition walletroles.Scenario
	require.NoError(t, yaml.Unmarshal(data, &definition))
	require.Len(t, definition.Steps, 7)
	require.Equal(t, "onboard-reference-wallet", definition.Steps[0]["id"])

	validation := definition.Steps[len(definition.Steps)-1]
	require.Equal(t, aggregate.ValidationStepID, validation["id"])
	with, ok := validation["with"].(map[string]any)
	require.True(t, ok)
	selectedTestIDs := walletroles.StringList(with["test_ids"])
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
	require.NoError(t, generateHappyFlow(testFCAFRoot, output))

	data, err := os.ReadFile(output)
	require.NoError(t, err)
	var definition walletroles.Scenario
	require.NoError(t, yaml.Unmarshal(data, &definition))
	require.Len(t, definition.Steps, 149)
	require.Equal(t, "onboard-reference-wallet", definition.Steps[0]["id"])

	validationSteps := make([]map[string]any, 0, 1)
	for index, step := range definition.Steps {
		if step["id"] == aggregate.ValidationStepID {
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
		walletroles.StringList(with["test_ids"]),
		"WS_RP_IA_MainInteraction__015",
		"happy flow must omit tests whose exact evidence source is not selected",
	)
	require.Len(t, walletroles.StringList(with["test_ids"]), 286)
	require.NotContains(
		t,
		walletroles.StringList(with["test_ids"]),
		"WS_RP_SM_RpIntegrity__006",
		"happy flow must omit tests whose exact evidence source is not selected",
	)
	require.NotContains(
		t,
		walletroles.StringList(with["test_ids"]),
		"WS_RP_SM_RpIntegrity__013",
		"happy flow must omit tests whose exact evidence source is not selected",
	)
	require.NotContains(
		t,
		walletroles.StringList(with["test_ids"]),
		"WS_RP_SM_RpIntegrity__016",
		"happy flow must omit tests without exact feasible evidence",
	)
	require.NotContains(
		t,
		walletroles.StringList(with["test_ids"]),
		"WS_RP_SM_RpIntegrity__018",
		"happy flow must omit tests without exact feasible evidence",
	)
	require.NotContains(
		t,
		walletroles.StringList(with["test_ids"]),
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
