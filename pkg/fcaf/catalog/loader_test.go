// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/validators"
	"github.com/stretchr/testify/require"
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

func TestLoadPreconditions(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "pre.yaml"), `
id: pipeline.pre-1
kind: pipeline
pipeline_id: org/pipeline
outputs:
  decoded_sdjwt:
    path: $.decoded
`)

	preconditions, err := LoadPreconditions(root)

	require.NoError(t, err)
	require.Contains(t, preconditions, "pipeline.pre-1")
}

func TestLoadGeneratedWalletRelyingPartyCatalog(t *testing.T) {
	cat, err := Load("../../../config_templates/fcaf/wallet_solution/relying_party")

	require.NoError(t, err)
	require.Len(t, cat.Tests, 172)
	require.Contains(t, cat.Preconditions, "pipeline.pid.presentation.sdjwt.all-claims")
	require.Contains(t, cat.Preconditions, "pipeline.pid.presentation.mdoc.all-claims-elements")
	require.Contains(
		t,
		cat.Preconditions,
		"assertion.pid.presentation.sdjwt.required-mandatory-claims-presented",
	)

	registry, err := validators.DefaultRegistry()
	require.NoError(t, err)
	for id, test := range cat.Tests {
		for _, assertion := range test.Assertions {
			_, exists := registry.Get(assertion.Validator)
			require.Truef(t, exists, "%s references unknown validator %s", id, assertion.Validator)
		}
	}
	for id, precondition := range cat.Preconditions {
		if precondition.Kind != "assertion" {
			continue
		}
		_, exists := registry.Get(precondition.Validator)
		require.Truef(t, exists, "%s references unknown validator %s", id, precondition.Validator)
	}

	selected, err := cat.ResolveSelectedTests(nil, "wallet_solution/relying_party", nil)
	require.NoError(t, err)
	require.Len(t, selected, 172)
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
preconditions:
  - ref: pipeline.pre-1
assertions:
  - id: claim-present
    validator: evidence.present
    input: evidence.decoded_sdjwt
verdict:
  pass_when: all_assertions_pass
`
}
