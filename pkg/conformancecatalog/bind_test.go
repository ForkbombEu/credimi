// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindParamsCheckKeysMatchColumns(t *testing.T) {
	t.Parallel()

	ch := Check{
		ID:         "id1",
		Path:       "fcaf/v1/suite/check.yaml",
		Title:      "Title",
		FSStandard: "fcaf",
		FSVersion:  "v1",
		Suite:      "suite",
		File:       "check.yaml",
		VisibleIn:  []string{"pipeline"},
		Protocol:   "oid4vp",
		SUT:        "wallet",
		Role:       "holder",
		Provider:   "forkbomb",
		Standard:   "FCAF",
		Component:  "wallet",
		Version:    "1.0",
	}
	overlay := dbx.Params{"created": "c", "updated": "u"}
	params, err := bindParams(checkColumns, ch, overlay)
	require.NoError(t, err)

	assertParamKeysMatchColumns(t, checkColumns, params)
	assert.Equal(t, ch.Path, params["path"])
	assert.Equal(t, ch.FSStandard, params["fs_standard"])
	assert.Equal(t, ch.Standard, params["standard"])
	assert.Equal(t, `["pipeline"]`, params["visible_in"])
	assert.Equal(t, "c", params["created"])
	assert.Equal(t, "u", params["updated"])
}

func TestBindParamsSuiteKeysMatchColumns(t *testing.T) {
	t.Parallel()

	s := SuiteRecord{
		ID:            "sid",
		Standard:      "FCAF",
		Component:     "wallet",
		ComponentRank: 0,
		Version:       "1.0",
		Suite:         "suite",
		Provider:      "forkbomb",
		SuiteName:     "Suite",
		CheckCount:    1,
		Members:       []SuiteMember{{Path: "a/b/c/d.yaml", Title: "T", File: "d.yaml"}},
		VisibleIn:     nil,
		FSStandard:    "fcaf",
		FSVersion:     "v1",
		PathPrefix:    "fcaf/v1/suite",
	}
	overlay := dbx.Params{"created": "c", "updated": "u"}
	params, err := bindParams(suiteColumns, s, overlay)
	require.NoError(t, err)

	assertParamKeysMatchColumns(t, suiteColumns, params)
	assert.Equal(t, "[]", params["visible_in"])
	assert.Equal(t, `[{"path":"a/b/c/d.yaml","title":"T","file":"d.yaml"}]`, params["members"])
	assert.Equal(t, 0, params["component_rank"])
	assert.Equal(t, 1, params["check_count"])
}

func TestBindParamsMissingColumnErrors(t *testing.T) {
	t.Parallel()

	cols := []columnSpec{
		{Name: "id", Kind: ColumnKindString},
		{Name: "created", Kind: ColumnKindString},
	}
	_, err := bindParams(cols, struct {
		ID string `json:"id"`
	}{ID: "x"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"created"`)
}

func assertParamKeysMatchColumns(t *testing.T, cols []columnSpec, p dbx.Params) {
	t.Helper()
	require.Len(t, p, len(cols))
	for _, c := range cols {
		_, ok := p[c.Name]
		require.True(t, ok, "missing bind key %q", c.Name)
	}
}
