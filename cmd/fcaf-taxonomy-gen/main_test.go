// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateTaxonomyIsCommitted(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	tmp := t.TempDir()

	source := filepath.Join(repoRoot, "pkg", "fcaf", "taxonomy", "taxonomy.json")
	raw, err := os.ReadFile(source)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "pkg", "fcaf", "taxonomy"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "webapp", "src", "lib", "fcaf"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, "pkg", "fcaf", "taxonomy", "taxonomy.json"),
		raw,
		0o644,
	))

	require.NoError(t, run(tmp))

	assertFileMatches(t,
		filepath.Join(tmp, "pkg", "fcaf", "taxonomy", "taxonomy_data.generated.go"),
		filepath.Join(repoRoot, "pkg", "fcaf", "taxonomy", "taxonomy_data.generated.go"),
	)

	// webapp/*.generated.* is gitignored (same as tests.generated.ts); assert the
	// generator still emits a non-empty TypeScript table for local/CI codegen.
	tsOut := filepath.Join(tmp, "webapp", "src", "lib", "fcaf", "taxonomy.generated.ts")
	tsData, err := os.ReadFile(tsOut)
	require.NoError(t, err)
	require.Contains(t, string(tsData), "FCAF_CATEGORY_ORDER")
	require.Contains(t, string(tsData), "FCAF_SUBGROUP_LABELS")
}

func assertFileMatches(t *testing.T, gotPath, wantPath string) {
	t.Helper()
	got, err := os.ReadFile(gotPath)
	require.NoError(t, err)
	want, err := os.ReadFile(wantPath)
	require.NoError(t, err)
	require.Equal(t, string(want), string(got), "%s is stale; run go run ./cmd/fcaf-taxonomy-gen", wantPath)
}
