// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package walletroles

import (
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/catalog"
	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/pocketbase/pocketbase/tools/inflector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const (
	repositoryRoot       = "../../../config_templates/fcaf"
	referenceNamespace   = "forkbomb-bv-andrea/eudiw-beta-wallet/"
	mobileAutomationTask = "mobile-automation"
	// screenshotStemBudget is PocketBase's 100-character cap on the snake-cased
	// `<step id>-<name>` stem, less the separator.
	screenshotStemBudget = 99
)

// shareTap counts the consent screen's Share action, which is the point at
// which the wallet spends a credential instance. A flow that shares twice, such
// as a scenario exercising two valid presentations in one step, spends two.
// A Share reached only through a `when: visible "Requested data"` guard counts
// too: the negative actions share whenever a tolerant Wallet shows the consent
// screen, and an instance that is spent but modelled as unspent would leave a
// later presentation of the same scenario running on an empty Wallet, whose
// refusal or no-match evidence would then prove nothing.
var shareTap = regexp.MustCompile(
	`tapOn:\s*(?:"Share"|\{[^}\n]*text:\s*"Share"|\n\s*text:\s*"Share")`,
)

var (
	flowParameter  = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)
	screenshotStep = regexp.MustCompile(
		`\$\{\{\s*([A-Za-z0-9_.-]+?)\.outputs\.flow_output\.maestro_screenshot_urls`,
	)
)

// suitePipeline is a suite's generated complete pipeline, with the roles each
// of its mobile steps runs.
type suitePipeline struct {
	suite           string
	steps           []map[string]any
	stepRoles       map[string][]string
	pipelineOutputs map[string]any
	tests           map[string]dsl.TestDefinition
}

func loadSuitePipeline(t *testing.T, suite string) suitePipeline {
	t.Helper()
	slug := strings.ReplaceAll(suite, "_", "-")
	slug = strings.ReplaceAll(slug, "/", "-")
	path := filepath.Join(
		repositoryRoot,
		suite,
		"pipelines",
		"fcaf-"+slug+"-complete-validation.yaml",
	)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var definition struct {
		Steps []map[string]any `yaml:"steps"`
	}
	require.NoError(t, yaml.Unmarshal(data, &definition), path)
	tests, err := catalog.LoadTests(filepath.Join(repositoryRoot, suite, "tests"))
	require.NoError(t, err)

	pipeline := suitePipeline{
		suite:     suite,
		steps:     definition.Steps,
		stepRoles: map[string][]string{},
		tests:     tests,
	}
	for _, step := range definition.Steps {
		id, _ := step["id"].(string)
		with, _ := step["with"].(map[string]any)
		switch use, _ := step["use"].(string); use {
		case mobileAutomationTask:
			pipeline.stepRoles[id] = stepRoles(t, id, with)
		case "fcaf-validation":
			pipeline.pipelineOutputs, _ = with["pipeline_outputs"].(map[string]any)
		}
	}
	require.NotEmpty(t, pipeline.pipelineOutputs, "%s has no fcaf-validation outputs", path)
	return pipeline
}

// stepRoles returns the roles a mobile step runs, in order; its own is last.
func stepRoles(t *testing.T, id string, with map[string]any) []string {
	t.Helper()
	var actionIDs []string
	if actionID, ok := with["action_id"].(string); ok {
		actionIDs = append(actionIDs, actionID)
	}
	entries, _ := with["actions"].([]any)
	for _, raw := range entries {
		entry, _ := raw.(map[string]any)
		actionID, _ := entry["id"].(string)
		actionIDs = append(actionIDs, actionID)
	}
	require.NotEmpty(t, actionIDs, "mobile step %q runs no stored action", id)
	roles := make([]string, 0, len(actionIDs))
	for _, actionID := range actionIDs {
		role, found := strings.CutPrefix(actionID, referenceNamespace)
		require.Truef(t, found, "mobile step %q runs %q, not a reference action", id, actionID)
		roles = append(roles, role)
	}
	return roles
}

// TestRepositoryContract pins fcaf-roles.yaml to the reference flows, the
// generated complete pipelines and the FCAF tests that read their screenshots.
func TestRepositoryContract(t *testing.T) {
	contract, err := Load(repositoryRoot)
	require.NoError(t, err)
	pipelines := []suitePipeline{
		loadSuitePipeline(t, "wallet_solution/relying_party"),
		loadSuitePipeline(t, "wallet_solution/attestation_provider"),
	}

	referenced := map[string][]string{}
	budgets := map[string]int{}
	for _, pipeline := range pipelines {
		for stepID, roles := range pipeline.stepRoles {
			budget := screenshotStemBudget - len(inflector.Snakecase(stepID))
			for _, role := range roles {
				if !slices.Contains(referenced[role], pipeline.suite) {
					referenced[role] = append(referenced[role], pipeline.suite)
				}
				if current, seen := budgets[role]; !seen || budget < current {
					budgets[role] = budget
				}
			}
		}
	}

	t.Run("role set", func(t *testing.T) {
		names := make([]string, 0, len(contract.Roles))
		for _, role := range contract.Roles {
			names = append(names, role.Name)
		}
		sort.Strings(names)
		assert.Equal(t, slices.Sorted(maps.Keys(referenced)), names)
	})

	t.Run("suites", func(t *testing.T) {
		for _, role := range contract.Roles {
			want := slices.Sorted(slices.Values(referenced[role.Name]))
			assert.Equalf(t, want, slices.Sorted(slices.Values(role.Suites)), "role %q", role.Name)
		}
	})

	t.Run("diagnostic budget", func(t *testing.T) {
		for _, role := range contract.Roles {
			assert.Equalf(
				t,
				budgets[role.Name],
				role.DiagnosticNameBudget,
				"role %q",
				role.Name,
			)
		}
	})

	t.Run("verdict bindings", func(t *testing.T) {
		owned := map[string]map[string]struct{}{}
		for _, pipeline := range pipelines {
			for _, binding := range verdictBindings(t, pipeline) {
				roles, mobile := pipeline.stepRoles[binding.stepID]
				require.Truef(
					t,
					mobile,
					"test %s reads %q from %q, not a mobile step",
					binding.testID,
					binding.name,
					binding.stepID,
				)
				role := roles[len(roles)-1]
				if owned[role] == nil {
					owned[role] = map[string]struct{}{}
				}
				owned[role][binding.name] = struct{}{}
				assert.LessOrEqualf(
					t,
					len(inflector.Snakecase(binding.name)),
					screenshotStemBudget-len(inflector.Snakecase(binding.stepID)),
					"test %s reads %q from step %q, whose stem PocketBase truncates",
					binding.testID,
					binding.name,
					binding.stepID,
				)
			}
		}
		for _, role := range contract.Roles {
			want := slices.Collect(maps.Keys(owned[role.Name]))
			got := make([]string, 0, len(role.Screenshots))
			for _, shot := range role.Screenshots {
				got = append(got, shot.Name)
			}
			assert.ElementsMatchf(t, want, got, "verdict screenshots of role %q", role.Name)
		}
	})

	t.Run("reference code", func(t *testing.T) {
		apps := map[string]App{}
		for app, packageID := range contract.Reference.AppIDs {
			apps[packageID] = app
		}
		for _, role := range contract.Roles {
			t.Run(role.Name, func(t *testing.T) {
				requireReferenceMatches(t, role, apps)
			})
		}
	})

	t.Run("injected roles", func(t *testing.T) {
		reset := referenceNamespace + contract.Injected(InjectedReset).Name
		for _, pipeline := range pipelines {
			prelude := pipeline.steps[0]
			require.Equal(t, "onboard-reference-wallet", prelude["id"], pipeline.suite)
			with, _ := prelude["with"].(map[string]any)
			assert.Equal(t, reset, with["action_id"], pipeline.suite)
		}
	})
}

type verdictBinding struct {
	testID string
	name   string
	stepID string
}

// verdictBindings returns every screenshot name a selected test of the suite
// reads, with the step that must store it.
func verdictBindings(t *testing.T, pipeline suitePipeline) []verdictBinding {
	t.Helper()
	var bindings []verdictBinding
	for _, testID := range slices.Sorted(maps.Keys(pipeline.tests)) {
		test := pipeline.tests[testID]
		for _, assertion := range test.Assertions {
			names := verdictNames(assertion)
			if len(names) == 0 {
				continue
			}
			evidence := strings.TrimPrefix(assertion.Input, "evidence.")
			binding, declared := test.Evidence[evidence]
			require.Truef(t, declared, "test %s has no evidence %q", testID, evidence)
			source, output, found := strings.Cut(binding.From, ".outputs.")
			require.Truef(t, found, "test %s evidence %q from %q", testID, evidence, binding.From)
			sourceDefinition, _ := pipeline.pipelineOutputs[source].(map[string]any)
			outputs, _ := sourceDefinition["output"].(map[string]any)
			value, selected := outputs[output]
			if !selected {
				continue
			}
			// A list of fixtures binds one issuance step per entry; each must
			// store the screenshot.
			encoded, err := yaml.Marshal(value)
			require.NoError(t, err)
			matches := screenshotStep.FindAllStringSubmatch(string(encoded), -1)
			require.NotEmptyf(
				t,
				matches,
				"test %s reads screenshots %v from %v, not a step's maestro_screenshot_urls",
				testID,
				names,
				value,
			)
			for _, match := range matches {
				for _, name := range names {
					bindings = append(
						bindings,
						verdictBinding{testID: testID, name: name, stepID: match[1]},
					)
				}
			}
		}
	}
	return bindings
}

// verdictNames returns the screenshot names an assertion reads.
func verdictNames(assertion dsl.AssertionDefinition) []string {
	param := func(key string, fallback string) string {
		if value, _ := assertion.Params[key].(string); value != "" {
			return value
		}
		return fallback
	}
	var names []string
	switch assertion.Validator {
	case "evidence.screenshot_named":
		names = append(names, param("name", ""))
	case "oid4vci.fixtures_stored":
		names = append(names, param("stored_screenshot", ""))
	case "oid4vci.issued_credential_presented":
		names = append(names, param("informed_screenshot", ""))
	case "oid4vci.offer_engagement":
		names = append(
			names,
			param("invoked_screenshot", "wallet-invoked"),
			param("not_invoked_screenshot", "wallet-not-invoked"),
		)
	}
	return slices.DeleteFunc(names, func(name string) bool { return name == "" })
}

func requireReferenceMatches(t *testing.T, role Role, apps map[string]App) {
	t.Helper()
	assert.NotEmpty(t, role.Category)

	var parameters []string
	for _, match := range flowParameter.FindAllStringSubmatch(role.ReferenceCode, -1) {
		if !slices.Contains(parameters, match[1]) {
			parameters = append(parameters, match[1])
		}
	}
	assert.ElementsMatch(t, role.Parameters, parameters, "parameters")

	var header struct {
		AppID string `yaml:"appId"`
	}
	require.NoError(t, yaml.NewDecoder(strings.NewReader(role.ReferenceCode)).Decode(&header))
	headerApp, known := apps[header.AppID]
	require.Truef(t, known, "header appId %q is not a reference app", header.AppID)
	assert.Equal(t, role.HeaderApp, headerApp, "header_app")

	commands, err := flowCommands(role.ReferenceCode)
	require.NoError(t, err)
	launched := map[App]struct{}{}
	for _, packageID := range launchedApps(commands, header.AppID) {
		app, known := apps[packageID]
		require.Truef(t, known, "launchApp %q is not a reference app", packageID)
		launched[app] = struct{}{}
	}
	assert.Equal(
		t,
		slices.Sorted(slices.Values(role.LaunchApps)),
		slices.Sorted(maps.Keys(launched)),
		"launch_apps",
	)

	assert.Equal(t, role.Presents, len(shareTap.FindAllString(role.ReferenceCode, -1)), "presents")

	conditions, err := ScreenshotConditions(role.ReferenceCode)
	require.NoError(t, err)
	if role.Silent {
		assert.Empty(t, conditions, "a silent role takes no screenshot")
	}
	for _, shot := range role.Screenshots {
		recorded, taken := conditions[shot.Name]
		if !assert.Truef(t, taken, "screenshot %q is never taken", shot.Name) {
			continue
		}
		for _, condition := range recorded {
			assert.Equalf(t, shot.When, condition, "screenshot %q", shot.Name)
		}
	}

	if role.Outcome == OutcomeWait {
		last := lastCommand(commands)
		wait, _ := last["extendedWaitUntil"].(map[string]any)
		require.NotNil(t, wait, "a wait role ends with extendedWaitUntil")
		assert.Equal(t, true, wait["optional"], "the final wait is optional")
		assert.Equal(t, role.WaitSeconds*1000, wait["timeout"], "the final wait lasts wait_seconds")
	}
}

// launchedApps returns the package of every launchApp at any depth; a launchApp
// without appId launches the header app.
func launchedApps(commands []any, headerApp string) []string {
	var packages []string
	for _, raw := range commands {
		switch command := raw.(type) {
		case string:
			if command == "launchApp" {
				packages = append(packages, headerApp)
			}
		case map[string]any:
			for key, value := range command {
				switch key {
				case "launchApp":
					packageID := headerApp
					switch options := value.(type) {
					case string:
						packageID = options
					case map[string]any:
						if appID, _ := options["appId"].(string); appID != "" {
							packageID = appID
						}
					}
					packages = append(packages, packageID)
				case "runFlow", "retry", "repeat":
					options, _ := value.(map[string]any)
					nested, _ := options["commands"].([]any)
					packages = append(packages, launchedApps(nested, headerApp)...)
				}
			}
		}
	}
	return packages
}

// lastCommand returns the last top-level command other than takeScreenshot.
func lastCommand(commands []any) map[string]any {
	for index := len(commands) - 1; index >= 0; index-- {
		command, _ := commands[index].(map[string]any)
		if _, screenshot := command["takeScreenshot"]; screenshot {
			continue
		}
		return command
	}
	return nil
}
