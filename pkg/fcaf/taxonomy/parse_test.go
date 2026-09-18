// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package taxonomy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

type parseCase struct {
	Name          string `json:"name"`
	TestID        string `json:"test_id"`
	Code          string `json:"code"`
	Subgroup      string `json:"subgroup"`
	Label         string `json:"label"`
	CategoryLabel string `json:"category_label"`
}

func TestParseGoldenCases(t *testing.T) {
	cases := loadParseCases(t)
	require.NotEmpty(t, cases)

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			parsed := Parse(tc.TestID)
			require.Equal(t, tc.Code, parsed.Code)
			require.Equal(t, tc.Subgroup, parsed.Subgroup)
			require.Equal(t, tc.Label, parsed.Label)
			require.Equal(t, tc.CategoryLabel, LabelFor(parsed.Code))
		})
	}
}

func TestCategoryOrderMatchesCategories(t *testing.T) {
	require.Equal(t, len(CategoryOrder), len(Categories))
	for _, code := range CategoryOrder {
		_, ok := Categories[code]
		require.Truef(t, ok, "CategoryOrder entry %q missing from Categories", code)
	}
}

func loadParseCases(t *testing.T) []parseCase {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	path := filepath.Join(filepath.Dir(file), "testdata", "parse_cases.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var cases []parseCase
	require.NoError(t, json.Unmarshal(raw, &cases))
	return cases
}
