// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadWalkIndexesRealFCAFTests(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "config_templates"))

	checks, err := LoadWalk(root)
	require.NoError(t, err)

	var fcaf []Check
	for _, ch := range checks {
		if ch.Standard == "fcaf" {
			fcaf = append(fcaf, ch)
		}
	}
	require.Greater(t, len(fcaf), 100, "expected hundreds of FCAF tests from config_templates")
	for _, ch := range fcaf {
		require.True(t, strings.HasPrefix(ch.Path, "fcaf/"), ch.Path)
		parts := strings.Split(ch.Path, "/")
		require.Len(t, parts, 4, "path identity fcaf/<version>/<suite>/<test_id>: %s", ch.Path)
		require.NotEmpty(t, parts[3], ch.Path)
		require.True(t, strings.HasSuffix(ch.File, ".yaml") || strings.HasSuffix(ch.File, ".yml"), ch.File)
		require.NotEmpty(t, ch.Title)
		require.Contains(t, ch.VisibleIn, SurfacePipeline)
	}
}
