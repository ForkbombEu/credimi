// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package aggregate

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/catalog"
	"github.com/forkbombeu/credimi/pkg/fcaf/walletroles"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const testSuite = "wallet_solution/relying_party"

// testFCAFRoot is the repository's FCAF templates root.
var testFCAFRoot = filepath.Join("..", "..", "..", "config_templates", "fcaf")

// buildComplete builds the relying-party complete pipeline of paths with
// the reference defaults changed by change.
func buildComplete(t *testing.T, paths []string, change func(*Options)) Result {
	t.Helper()
	opts := Options{
		Suite:   testSuite,
		Name:    "FCAF wallet relying-party complete validation",
		NoVideo: new(true),
	}
	if change != nil {
		change(&opts)
	}
	built, err := Build(testFCAFRoot, paths, opts)
	require.NoError(t, err)
	return built
}

func testScenarioPaths(t *testing.T) []string {
	t.Helper()
	paths, err := ScenarioPaths(testFCAFRoot, testSuite)
	require.NoError(t, err)
	return paths
}

func TestBuildWithDefaultsIsTheCommittedPipeline(t *testing.T) {
	built := buildComplete(t, testScenarioPaths(t), nil)
	committed, err := os.ReadFile(filepath.Join(
		testFCAFRoot,
		testSuite,
		"pipelines",
		"fcaf-wallet-solution-relying-party-complete-validation.yaml",
	))
	require.NoError(t, err)
	require.Equal(t, string(committed), string(built.YAML))
	require.Empty(t, built.Skipped)
	require.Len(t, built.TestIDs, 617)
}

// testActionIDs maps every role of the contract to an action path made by
// path(role).
func testActionIDs(t *testing.T, path func(role string) string) map[string]string {
	t.Helper()
	contract, err := walletroles.Load(testFCAFRoot)
	require.NoError(t, err)
	actions := make(map[string]string, len(contract.Roles))
	for _, role := range contract.Roles {
		actions[role.Name] = path(role.Name)
	}
	return actions
}

// TestBuildMovesEveryWalletActionToTheWalletsAction pins that ActionIDs
// replaces every reference wallet action id, injected resets and issuances
// included, with the wallet's action of the role, whatever its path, and
// changes no other string; VersionID and DeviceID set the prelude version
// and the device.
func TestBuildMovesEveryWalletActionToTheWalletsAction(t *testing.T) {
	paths := testScenarioPaths(t)
	roles := testWalletRoles(t)
	// A caller's action on another organization's wallet may carry a suffix.
	actions := testActionIDs(t, func(role string) string {
		return "owner/acme-wallet/" + role + "-1"
	})
	reference := buildComplete(t, paths, nil)
	moved := buildComplete(t, paths, func(o *Options) { o.ActionIDs = actions })
	require.Equal(t, reference.TestIDs, moved.TestIDs)

	var want, got map[string]any
	require.NoError(t, yaml.Unmarshal(reference.YAML, &want))
	require.NoError(t, yaml.Unmarshal(moved.YAML, &got))
	changed := 0
	var walk func(path string, want, got any)
	walk = func(path string, want, got any) {
		switch typed := want.(type) {
		case map[string]any:
			other, ok := got.(map[string]any)
			require.True(t, ok, path)
			require.ElementsMatch(
				t,
				slices.Collect(maps.Keys(typed)),
				slices.Collect(maps.Keys(other)),
				path,
			)
			for key, value := range typed {
				walk(path+"."+key, value, other[key])
			}
		case []any:
			other, ok := got.([]any)
			require.True(t, ok, path)
			require.Len(t, other, len(typed), path)
			for index, value := range typed {
				walk(fmt.Sprintf("%s[%d]", path, index), value, other[index])
			}
		case string:
			if typed == got {
				return
			}
			require.True(
				t,
				strings.HasSuffix(path, ".with.action_id") ||
					regexp.MustCompile(`\.with\.actions\[\d+\]\.id$`).MatchString(path),
				"%s changed from %q to %q",
				path, typed, got,
			)
			require.Equal(t, actions[strings.TrimPrefix(typed, roles.prefix)], got, path)
			changed++
		default:
			require.Equal(t, typed, got, path)
		}
	}
	walk("", want, got)
	require.Positive(t, changed)

	var definition walletroles.Scenario
	require.NoError(t, yaml.Unmarshal(moved.YAML, &definition))
	for _, step := range definition.Steps {
		for _, id := range walletActionIDs(step) {
			require.True(t, slices.Contains(slices.Collect(maps.Values(actions)), id),
				"step %v runs %s", step["id"], id)
		}
	}

	placed := buildComplete(t, paths, func(o *Options) {
		o.ActionIDs = actions
		o.VersionID = "acme/acme-wallet/1-0"
		o.DeviceID = "acme/runner/pixel"
	})
	var placedDefinition walletroles.Scenario
	require.NoError(t, yaml.Unmarshal(placed.YAML, &placedDefinition))
	require.Equal(t, "acme/runner/pixel", placedDefinition.Runtime["global_device_id"])
	prelude, _ := placedDefinition.Steps[0]["with"].(map[string]any)
	require.Equal(t, PreludeStepID, placedDefinition.Steps[0]["id"])
	require.Equal(t, "acme/acme-wallet/1-0", prelude["version_id"])
	require.Equal(t, actions["fcaf-reset-and-onboard"], prelude["action_id"])
}

// TestBuildLeavesOutTheScenariosOfAMissingRole pins that a scenario needing
// a role the wallet lacks is left out whole, with its tests, and listed:
// the pipeline is the one of the other scenarios alone.
func TestBuildLeavesOutTheScenariosOfAMissingRole(t *testing.T) {
	paths := testScenarioPaths(t)
	roles := testWalletRoles(t)

	usedBy := map[string][]string{}
	for _, path := range paths {
		scenario, err := walletroles.LoadScenario(path)
		require.NoError(t, err)
		for _, step := range scenario.Steps {
			for _, id := range walletActionIDs(step) {
				name := strings.TrimPrefix(id, roles.prefix)
				if !slices.Contains(usedBy[name], path) {
					usedBy[name] = append(usedBy[name], path)
				}
			}
		}
	}
	// The role run by the fewest scenarios that still run another one.
	missing := ""
	for name, users := range usedBy {
		// Injected roles also run where the scenarios do not name them.
		if roles.prefix+name == roles.resetActionID ||
			roles.prefix+name == roles.issuanceActionID || len(users) == len(paths) {
			continue
		}
		if missing == "" || len(users) < len(usedBy[missing]) ||
			len(users) == len(usedBy[missing]) && name < missing {
			missing = name
		}
	}
	require.NotEmpty(t, missing)
	// The wallet's actions are the reference ones, so the pipeline of the
	// other scenarios is the reference one.
	have := testActionIDs(t, func(role string) string { return roles.prefix + role })
	delete(have, missing)

	built := buildComplete(t, paths, func(o *Options) { o.ActionIDs = have })
	skipped := make([]SkippedScenario, 0, len(usedBy[missing]))
	kept := make([]string, 0, len(paths))
	for _, path := range paths {
		if slices.Contains(usedBy[missing], path) {
			skipped = append(skipped, SkippedScenario{
				Scenario:     filepath.Base(path),
				MissingRoles: []string{missing},
			})
			continue
		}
		kept = append(kept, path)
	}
	require.Equal(t, skipped, built.Skipped)
	alone := buildComplete(t, kept, nil)
	require.Equal(t, string(alone.YAML), string(built.YAML))
	require.Equal(t, alone.TestIDs, built.TestIDs)
	require.Less(t, len(built.TestIDs), 617)

	// A wallet's build filters its tests by evidence: with every role it
	// still has every test.
	every := testActionIDs(t, func(role string) string { return roles.prefix + role })
	filtered := buildComplete(t, paths, func(o *Options) {
		o.ActionIDs, o.FilterTestsByEvidence = every, true
	})
	require.Empty(t, filtered.Skipped)
	require.Equal(t, buildComplete(t, paths, nil).TestIDs, filtered.TestIDs)

	// Without the engagement role, no kept test reads evidence of a skipped
	// scenario.
	withoutEngagement := maps.Clone(every)
	delete(withoutEngagement, "fcaf-engagement-haip-vp")
	partial := buildComplete(t, paths, func(o *Options) {
		o.ActionIDs, o.FilterTestsByEvidence = withoutEngagement, true
	})
	require.NotEmpty(t, partial.Skipped)
	skippedPrefixes := make([]string, 0, len(partial.Skipped))
	for _, scenario := range partial.Skipped {
		skippedPrefixes = append(skippedPrefixes, ScenarioPrefix(testSuite, scenario.Scenario)+"-")
	}
	definitions, err := catalog.LoadTests(filepath.Join(testFCAFRoot, testSuite, "tests"))
	require.NoError(t, err)
	for _, id := range partial.TestIDs {
		for name, binding := range definitions[id].Evidence {
			source, _, _ := strings.Cut(binding.From, ".outputs.")
			for _, prefix := range skippedPrefixes {
				require.Falsef(
					t,
					strings.HasPrefix(source, prefix),
					"test %s reads evidence %s of the skipped step %s",
					id,
					name,
					source,
				)
			}
		}
	}
	require.Less(t, len(partial.TestIDs), len(filtered.TestIDs))

	// Without the reset role no scenario driving the wallet can run.
	delete(have, strings.TrimPrefix(roles.resetActionID, roles.prefix))
	_, err = Build(testFCAFRoot, usedBy[missing][:1], Options{Suite: testSuite, ActionIDs: have})
	require.ErrorIs(t, err, ErrNothingToRun)
}

// TestAggregateHoldsACredentialForEveryPresentation guards the reference
// wallet's single-use credential instances and the per-scenario wallet reset:
// every wallet action runs after its own scenario's reset, a presentation that
// can reach Share is preceded by an unspent issuance of the format it requests
// from the same scenario, and a presentation the wallet refuses still finds a
// PID of that format, so the refusal is not just an empty wallet. An absence
// probe is the exception: it checks that the wallet holds nothing matching a
// credential it had to refuse, so no PID may be issued for it. A step's
// `actions` run in order, so each entry counts as a wallet step of its own.
func TestAggregateHoldsACredentialForEveryPresentation(t *testing.T) {
	root := filepath.Join(testFCAFRoot, "wallet_solution", "relying_party")
	roles, err := loadWalletRoles(testFCAFRoot)
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
		var definition walletroles.Scenario
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
		absenceProbes := 0
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
				if actionID == roles.resetActionID {
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
				if issuesCredential(action, roles) {
					available[issuedCredentialFormat(source)]++
					if actionID != roles.issuanceActionID {
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
				consumed, err := credentialInstancesConsumed(action, roles)
				require.NoError(t, err)
				shares += consumed
				// A scenario-authored issuance may deliberately stay unspent, for
				// example to prove a held credential does not match the request,
				// but every Share must find an unspent instance of the requested
				// format.
				for _, format := range presentedCredentialFormats(source, consumed, available) {
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
					for _, format := range formatsSpentEntirely(source, available) {
						available[format] = 0
					}
				}
				if consumed == 0 && opensPresentationRequest(action, source) {
					if probesAbsence(action, roles) {
						require.Falsef(
							t,
							slices.ContainsFunc(definition.Steps, func(other map[string]any) bool {
								return other["id"] == id+"-issue-pid-session"
							}),
							"%s: absence probe %q gets an injected PID",
							pipeline,
							id,
						)
						absenceProbes++
						continue
					}
					for _, format := range presentedCredentialFormats(source, 1, available) {
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
			"%s: %d resets, %d shares, %d injected issuances, %d absence probes",
			pipeline,
			resets,
			shares,
			injected,
			absenceProbes,
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
		require.NotContains(t, entry, "code", "step %v", step["id"])
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
