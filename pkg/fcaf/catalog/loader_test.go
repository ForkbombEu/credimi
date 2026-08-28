// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/validators"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestLoadTestsByID(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "test.yaml"), testYAML())

	tests, err := LoadTestsByID(root, []string{"test-1"})

	require.NoError(t, err)
	require.Contains(t, tests, "test-1")
}

func TestLoadTestsRejectsDuplicates(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.yaml"), testYAML())
	writeTestFile(t, filepath.Join(root, "nested", "b.yaml"), testYAML())

	_, err := LoadTests(root)

	require.ErrorContains(t, err, "duplicate fcaf test id")
}

func TestLoadTestsSkipsImplementationFolder(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.yaml"), testYAML())
	writeTestFile(t, filepath.Join(root, "_implementation", "note.yaml"), "not: a test\n")

	tests, err := LoadTests(root)

	require.NoError(t, err)
	require.Len(t, tests, 1)
}

func TestLoadGeneratedWalletRelyingPartyCatalog(t *testing.T) {
	cat, err := Load("../../../config_templates/fcaf/wallet_solution/relying_party")

	require.NoError(t, err)
	require.Len(t, cat.Tests, 559)

	registry, err := validators.DefaultRegistry()
	require.NoError(t, err)
	for id, test := range cat.Tests {
		for name, binding := range test.Evidence {
			require.Containsf(t, binding.From, ".outputs.", "%s evidence %s", id, name)
		}
		for _, assertion := range test.Assertions {
			_, exists := registry.Get(assertion.Validator)
			require.Truef(t, exists, "%s references unknown validator %s", id, assertion.Validator)
		}
	}

	selected, err := cat.ResolveSelectedTests(nil, "wallet_solution/relying_party", nil)
	require.NoError(t, err)
	require.Len(t, selected, 559)
}

func TestEmbeddedValidationStepsExposeDirectTestEvidence(t *testing.T) {
	catalogRoot := "../../../config_templates/fcaf/wallet_solution/relying_party"
	cat, err := Load(catalogRoot)
	require.NoError(t, err)

	type validationStep struct {
		Use  string `yaml:"use"`
		With struct {
			TestID          string   `yaml:"test_id"`
			TestIDs         []string `yaml:"test_ids"`
			PipelineOutputs map[string]struct {
				Output map[string]any `yaml:"output"`
			} `yaml:"pipeline_outputs"`
		} `yaml:"with"`
	}
	var definitions struct {
		Steps []validationStep `yaml:"steps"`
	}

	paths, err := filepath.Glob(filepath.Join(catalogRoot, "pipelines", "*.yaml"))
	require.NoError(t, err)
	validationSteps := 0
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		definitions.Steps = nil
		require.NoError(t, yaml.Unmarshal(data, &definitions), path)
		for _, step := range definitions.Steps {
			if step.Use != "fcaf-validation" {
				continue
			}
			validationSteps++
			testIDs := append([]string(nil), step.With.TestIDs...)
			if step.With.TestID != "" {
				testIDs = append(testIDs, step.With.TestID)
			}
			for _, testID := range testIDs {
				test, exists := cat.Tests[testID]
				require.Truef(t, exists, "%s references unknown test %s", path, testID)
				for name, binding := range test.Evidence {
					parts := strings.SplitN(binding.From, ".outputs.", 2)
					require.Lenf(t, parts, 2, "%s evidence %s", testID, name)
					pipeline, exists := step.With.PipelineOutputs[parts[0]]
					require.Truef(
						t,
						exists,
						"%s does not expose source %s required by %s evidence %s",
						path,
						parts[0],
						testID,
						name,
					)
					_, exists = pipeline.Output[parts[1]]
					require.Truef(
						t,
						exists,
						"%s source %s does not expose %s required by %s evidence %s",
						path,
						parts[0],
						parts[1],
						testID,
						name,
					)
				}
			}
		}
	}
	require.Positive(t, validationSteps)
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func testYAML() string {
	return `
id: test-1
title: Test title
source:
  path: source.md
suite:
  sut: wallet_solution
  role: relying_party
  section: data_model.address_data
applicability:
  credential_format: sd-jwt-vc
normative_references:
  - title: reference
evidence:
  decoded_sdjwt:
    from: pipeline.pid.outputs.decoded_sdjwt
assertions:
  - id: claim-present
    validator: evidence.present
    input: evidence.decoded_sdjwt
verdict:
  pass_when: all_assertions_pass
`
}
