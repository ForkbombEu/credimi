// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package main generates complete and demo FCAF aggregate pipelines.
package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const validationTask = "fcaf-validation"

type pipelineDefinition struct {
	Name    string           `yaml:"name"`
	Runtime map[string]any   `yaml:"runtime,omitempty"`
	Steps   []map[string]any `yaml:"steps"`
}

type aggregateDefinition struct {
	Name    string           `yaml:"name"`
	Runtime map[string]any   `yaml:"runtime"`
	Steps   []map[string]any `yaml:"steps"`
}

var nonSlugCharacter = regexp.MustCompile(`[^a-z0-9]+`)

const (
	completeValidationName  = "FCAF wallet relying-party complete validation"
	demoValidationName      = "FCAF wallet relying-party demo validation"
	happyFlowValidationName = "FCAF wallet relying-party happy flow validation"
)

var demoScenarioNames = []string{
	"fcaf-wallet-solution-relying-party-engagement-haip-vp.yaml",
}

// demoTestIDs keeps the showcase bounded to one proven HAIP-VP happy flow.
var demoTestIDs = []string{
	"WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_003",
	"WS_RP_DM_AddressData_Mobilephonenumber_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_AddressData_Mobilephonenumber_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_AddressData_Mobilephonenumber_PID_IETF-sd-jwt-vc_003",
	"WS_RP_DM_AddressData_Residentaddress_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_AddressData_Residentaddress_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_AddressData_Residentcity_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_AddressData_Residentcity_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_AddressData_Residentcountry_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_AddressData_Residentcountry_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_AddressData_Residentcountry_PID_IETF-sd-jwt-vc_003",
	"WS_RP_DM_AddressData_Residenthousenumber_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_AddressData_Residenthousenumber_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_AddressData_Residentpostalcode_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_AddressData_Residentpostalcode_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_AddressData_Residentstate_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_AddressData_Residentstate_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_AddressData_Residentstreet_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_AddressData_Residentstreet_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_Credentialmetadata_Documentnumber_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_Credentialmetadata_Documentnumber_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_Credentialmetadata_Expirydate_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_Credentialmetadata_Expirydate_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_Credentialmetadata_Expirydate_PID_IETF-sd-jwt-vc_003",
	"WS_RP_DM_Credentialmetadata_Expirydate_PID_IETF-sd-jwt-vc_004",
	"WS_RP_DM_Credentialmetadata_Issuancedate_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_Credentialmetadata_Issuancedate_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_Credentialmetadata_Issuancedate_PID_IETF-sd-jwt-vc_003",
	"WS_RP_DM_Credentialmetadata_Issuancedate_PID_IETF-sd-jwt-vc_004",
	"WS_RP_DM_Credentialmetadata_Issuingauthority_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_Credentialmetadata_Issuingauthority_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_Credentialmetadata_Issuingcountry_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_Credentialmetadata_Issuingcountry_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_Credentialmetadata_Issuingcountry_PID_IETF-sd-jwt-vc_003",
	"WS_RP_DM_Credentialmetadata_Issuingjurisdiction_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_Credentialmetadata_Issuingjurisdiction_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_Credentialmetadata_Issuingjurisdiction_PID_IETF-sd-jwt-vc_003",
	"WS_RP_DM_IdentifyingData_Familyname_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_IdentifyingData_Familyname_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_IdentifyingData_Givenname_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_IdentifyingData_Givenname_PID_IETF-sd-jwt-vc_002",
	"WS_RP_IA_Engagement__001",
	"WS_RP_SM_DeviceBinding__007",
	"WS_RP_SM_IssuerIntegrity__013",
}

func main() {
	inputDir := flag.String(
		"input",
		"config_templates/fcaf/wallet_solution/relying_party/scenarios",
		"directory containing FCAF scenario pipeline YAML",
	)
	outputPath := flag.String(
		"output",
		"config_templates/fcaf/wallet_solution/relying_party/pipelines/fcaf-wallet-solution-relying-party-complete-validation.yaml",
		"generated complete validation pipeline YAML",
	)
	demoOutputPath := flag.String(
		"demo-output",
		"config_templates/fcaf/wallet_solution/relying_party/pipelines/fcaf-wallet-solution-relying-party-demo-validation.yaml",
		"generated demo validation pipeline YAML",
	)
	happyFlowOutputPath := flag.String(
		"happy-flow-output",
		"config_templates/fcaf/wallet_solution/relying_party/pipelines/fcaf-wallet-solution-relying-party-happy-flow-validation.yaml",
		"generated happy flow validation pipeline YAML",
	)
	flag.Parse()

	if err := generate(*inputDir, *outputPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := generateDemo(*inputDir, *demoOutputPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := generateHappyFlow(*inputDir, *happyFlowOutputPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generateDemo(inputDir string, outputPath string) error {
	selected, err := selectScenarios(inputDir, demoScenarioNames)
	if err != nil {
		return err
	}
	return buildAggregate(demoValidationName, selected, outputPath, demoTestIDs)
}

// happyFlowScenarioNames keeps the happy flow to the shared-evidence
// scenarios that own the positive test batches; the fragmented one-test
// DCQL variants stay in the complete validation aggregate only.
var happyFlowScenarioNames = []string{
	"fcaf-wallet-solution-relying-party-engagement-haip-vp.yaml",
	"fcaf-wallet-solution-relying-party-pid-mdoc-data-model.yaml",
	"fcaf-wallet-solution-relying-party-dcql-protocol-messages.yaml",
	"fcaf-wallet-solution-relying-party-dcql-metadata.yaml",
	"fcaf-wallet-solution-relying-party-dcql-main-interaction.yaml",
	"fcaf-wallet-solution-relying-party-dcql-rp-integrity.yaml",
	"fcaf-wallet-solution-relying-party-dcql-credential-formats.yaml",
	"fcaf-wallet-solution-relying-party-dcql-trust-mechanisms.yaml",
	"fcaf-wallet-solution-relying-party-dcql-session-encryption.yaml",
	"fcaf-wallet-solution-relying-party-dcql-interaction-metadata.yaml",
	"fcaf-wallet-solution-relying-party-request-object-by-value.yaml",
	"fcaf-wallet-solution-relying-party-dcql-cryptography.yaml",
	"fcaf-wallet-solution-relying-party-dcql-credentials-match.yaml",
	"fcaf-wallet-solution-relying-party-dcql-device-binding.yaml",
}

// generateHappyFlow builds the aggregate pipeline validating every FCAF test
// owned by the single happy-flow scenario, without the curated demo filter.
func generateHappyFlow(inputDir string, outputPath string) error {
	selected, err := selectScenarios(inputDir, happyFlowScenarioNames)
	if err != nil {
		return err
	}
	return buildAggregate(happyFlowValidationName, selected, outputPath, nil)
}

func selectScenarios(inputDir string, names []string) ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(inputDir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("find FCAF scenarios: %w", err)
	}
	selected := make([]string, 0, len(names))
	for _, path := range paths {
		for _, name := range names {
			if filepath.Base(path) == name {
				selected = append(selected, path)
				break
			}
		}
	}
	if len(selected) != len(names) {
		return nil, fmt.Errorf(
			"FCAF scenarios incomplete: expected %v, found %d files",
			names,
			len(selected),
		)
	}
	sort.Strings(selected)
	return selected, nil
}

func generate(inputDir string, outputPath string) error {
	paths, err := filepath.Glob(filepath.Join(inputDir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("find FCAF scenarios: %w", err)
	}
	if len(paths) == 0 {
		return fmt.Errorf("no FCAF scenarios found in %s", inputDir)
	}
	sort.Strings(paths)
	return buildAggregate(completeValidationName, paths, outputPath, nil)
}

func buildAggregate(
	name string,
	paths []string,
	outputPath string,
	allowedTestIDs []string,
) error {
	var allowedTests map[string]struct{}
	if allowedTestIDs != nil {
		allowedTests = make(map[string]struct{}, len(allowedTestIDs))
		for _, id := range allowedTestIDs {
			if strings.TrimSpace(id) == "" {
				return fmt.Errorf("requested FCAF test ID is empty")
			}
			if _, exists := allowedTests[id]; exists {
				return fmt.Errorf("requested FCAF test %q is duplicated", id)
			}
			allowedTests[id] = struct{}{}
		}
	}

	aggregate := aggregateDefinition{
		Name: name,
		Runtime: map[string]any{
			"global_device_id": "forkbomb-bv-andrea/usb/device",
			"temporal": map[string]any{
				"activity_options": map[string]any{
					"schedule_to_close_timeout": "50m",
					"start_to_close_timeout":    "50m",
					"heartbeat_timeout":         "10m",
					"retry_policy": map[string]any{
						"maximum_attempts": 1,
					},
				},
			},
		},
		Steps: make([]map[string]any, 0, len(paths)*4),
	}
	aggregate.Steps = append(aggregate.Steps, onboardingPrelude())
	pipelineOutputs := map[string]any{}
	testIDs := map[string]struct{}{}
	seenStepIDs := map[string]string{}

	for _, path := range paths {
		definition, err := loadPipeline(path)
		if err != nil {
			return err
		}
		prefix := scenarioPrefix(path)
		fixture := nestedMap(definition.Runtime, "fixture")
		stepIDs, err := scenarioStepIDs(path, definition.Steps, prefix)
		if err != nil {
			return err
		}

		for _, step := range definition.Steps {
			use, _ := step["use"].(string)
			if use == validationTask {
				if err := mergeValidationStep(
					path,
					step,
					stepIDs,
					fixture,
					pipelineOutputs,
					testIDs,
					allowedTests,
				); err != nil {
					return err
				}
				continue
			}

			rewritten := rewriteValue(step, stepIDs, fixture).(map[string]any)
			rewritten["continue_on_error"] = true
			id, _ := rewritten["id"].(string)
			if previous, exists := seenStepIDs[id]; exists {
				return fmt.Errorf(
					"generated step id %q collides between %s and %s",
					id,
					previous,
					path,
				)
			}
			seenStepIDs[id] = path
			aggregate.Steps = append(aggregate.Steps, rewritten)
		}
	}

	for _, id := range allowedTestIDs {
		if _, selected := testIDs[id]; !selected {
			return fmt.Errorf("requested FCAF test %q is not owned by selected scenarios", id)
		}
	}

	selectedTests := make([]string, 0, len(testIDs))
	for id := range testIDs {
		selectedTests = append(selectedTests, id)
	}
	sort.Strings(selectedTests)
	if len(selectedTests) == 0 {
		return fmt.Errorf("FCAF scenarios select no tests")
	}

	aggregate.Steps = append(aggregate.Steps, map[string]any{
		"id":  "run-complete-fcaf-validation",
		"use": validationTask,
		"with": map[string]any{
			"test_ids":         selectedTests,
			"suite":            "wallet_solution/relying_party",
			"pipeline_outputs": pipelineOutputs,
		},
	})

	data, err := yaml.Marshal(aggregate)
	if err != nil {
		return fmt.Errorf("encode aggregate FCAF pipeline: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create aggregate pipeline directory: %w", err)
	}
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("write aggregate FCAF pipeline: %w", err)
	}
	return nil
}

func loadPipeline(path string) (pipelineDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return pipelineDefinition{}, fmt.Errorf("read FCAF scenario %s: %w", path, err)
	}
	var definition pipelineDefinition
	if err := yaml.Unmarshal(data, &definition); err != nil {
		return pipelineDefinition{}, fmt.Errorf("parse FCAF scenario %s: %w", path, err)
	}
	if strings.TrimSpace(definition.Name) == "" || len(definition.Steps) == 0 {
		return pipelineDefinition{}, fmt.Errorf("FCAF scenario %s must define name and steps", path)
	}
	return definition, nil
}

func onboardingPrelude() map[string]any {
	return map[string]any{
		"id":  "onboard-reference-wallet",
		"use": "mobile-automation",
		"with": map[string]any{
			"action_id":  "forkbomb-bv-andrea/eudiw-beta-wallet/onboarding-1",
			"version_id": "forkbomb-bv-andrea/eudiw-beta-wallet/2026-06-38-demo",
		},
	}
}

func scenarioStepIDs(
	path string,
	steps []map[string]any,
	prefix string,
) (map[string]string, error) {
	ids := map[string]string{}
	for _, step := range steps {
		use, _ := step["use"].(string)
		if use == validationTask {
			continue
		}
		id, _ := step["id"].(string)
		if strings.TrimSpace(id) == "" {
			return nil, fmt.Errorf("FCAF scenario %s contains a step without id", path)
		}
		if _, exists := ids[id]; exists {
			return nil, fmt.Errorf("FCAF scenario %s contains duplicate step id %q", path, id)
		}
		ids[id] = prefix + "-" + id
	}
	return ids, nil
}

func mergeValidationStep(
	path string,
	step map[string]any,
	stepIDs map[string]string,
	fixture map[string]any,
	pipelineOutputs map[string]any,
	testIDs map[string]struct{},
	allowedTestIDs map[string]struct{},
) error {
	with, ok := step["with"].(map[string]any)
	if !ok {
		return fmt.Errorf("FCAF validation step in %s has no with object", path)
	}
	addTestID := func(id string) {
		if id == "" {
			return
		}
		if allowedTestIDs != nil {
			if _, allowed := allowedTestIDs[id]; !allowed {
				return
			}
		}
		testIDs[id] = struct{}{}
	}
	for _, id := range stringSlice(with["test_ids"]) {
		addTestID(id)
	}
	if id, _ := with["test_id"].(string); id != "" {
		addTestID(id)
	}
	rawOutputs, ok := with["pipeline_outputs"].(map[string]any)
	if !ok || len(rawOutputs) == 0 {
		return fmt.Errorf("FCAF validation step in %s has no pipeline_outputs", path)
	}
	rewritten := optionalizeExpressions(
		rewriteValue(rawOutputs, stepIDs, fixture),
	).(map[string]any)
	for source, output := range rewritten {
		if previous, exists := pipelineOutputs[source]; exists {
			return fmt.Errorf(
				"FCAF evidence source %q is defined by multiple scenarios (%T and %s)",
				source,
				previous,
				path,
			)
		}
		pipelineOutputs[source] = output
	}
	return nil
}

func rewriteValue(value any, stepIDs map[string]string, fixture map[string]any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = rewriteValue(child, stepIDs, fixture)
		}
		if id, ok := out["id"].(string); ok {
			if rewritten, exists := stepIDs[id]; exists {
				out["id"] = rewritten
			}
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for index, child := range typed {
			out[index] = rewriteValue(child, stepIDs, fixture)
		}
		return out
	case string:
		return rewriteString(typed, stepIDs, fixture)
	default:
		return value
	}
}

func optionalizeExpressions(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = optionalizeExpressions(child)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for index, child := range typed {
			out[index] = optionalizeExpressions(child)
		}
		return out
	case string:
		trimmed := strings.TrimSpace(typed)
		if strings.HasPrefix(trimmed, "${{") && strings.HasSuffix(trimmed, "}}") {
			inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "${{"), "}}"))
			if !strings.Contains(inner, "| optional") {
				return "${{ " + inner + " | optional }}"
			}
		}
		return typed
	default:
		return value
	}
}

func rewriteString(value string, stepIDs map[string]string, fixture map[string]any) string {
	keys := make([]string, 0, len(stepIDs))
	for key := range stepIDs {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	for _, oldID := range keys {
		value = strings.ReplaceAll(value, "${{ "+oldID+".", "${{ "+stepIDs[oldID]+".")
		if value == oldID {
			value = stepIDs[oldID]
		}
	}
	for key, raw := range fixture {
		replacement, ok := raw.(string)
		if !ok {
			continue
		}
		value = strings.ReplaceAll(value, "${fixture."+key+"}", replacement)
	}
	return value
}

func nestedMap(values map[string]any, key string) map[string]any {
	if values == nil {
		return nil
	}
	value, _ := values[key].(map[string]any)
	return value
}

func stringSlice(value any) []string {
	switch typed := value.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	case []string:
		return typed
	default:
		return nil
	}
}

func scenarioPrefix(path string) string {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	name = strings.TrimPrefix(name, "fcaf-wallet-solution-relying-party-")
	name = nonSlugCharacter.ReplaceAllString(strings.ToLower(name), "-")
	name = strings.Trim(name, "-")
	if len(name) > 36 {
		name = strings.Trim(name[:36], "-")
	}
	sum := sha256.Sum256([]byte(filepath.Base(path)))
	return fmt.Sprintf("%s-%x", name, sum[:4])
}
