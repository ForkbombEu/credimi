// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package canonify

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// A simple mock DB that holds existing names in a map.
type mockDB struct {
	existing map[string]struct{}
}

func (m *mockDB) Exists(name string) bool {
	_, ok := m.existing[name]
	return ok
}

func makeMock(names ...string) *mockDB {
	db := &mockDB{existing: make(map[string]struct{})}
	for _, n := range names {
		db.existing[n] = struct{}{}
	}
	return db
}

func TestCanonifyAllCases(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		want          string
		existing      []string
		NotExistsFunc bool
		options       CanonifyOptions
		expectErr     bool
	}{
		// === Basic ASCII ===
		{name: "basic ascii", input: "Hello World!", want: "hello_world"},
		{
			name:  "multiple spaces",
			input: "   Multiple   Spaces\tand\nTabs  ",
			want:  "multiple_spaces_and_tabs",
		},
		{name: "punctuation removed", input: "100% fun!", want: "100_fun"},
		{name: "email-like input", input: "user@domain.com", want: "user_domain_com"},
		{name: "weird symbols", input: "--weird__chars++", want: "weird_chars"},
		{name: "mixed case with numbers", input: "miXED1323", want: "mixed1323"},
		// === Accents / diacritics ===
		{name: "accents removed", input: "ÁÉÍÓÚ Ñ ç", want: "aeiou_n_c"},

		// === Invisible / zero-width characters ===
		{name: "zero width chars", input: "A\u200B\u200D\uFEFF\u200C\u2060B", want: "ab"},
		{name: "fallback if empty", input: "\u200B\u200D\u2060", want: "fallback",
			options: CanonifyOptions{
				Separator:   '_',
				MinLen:      1,
				Fallback:    "fallback",
				MaxAttempts: 10,
			}},
		{
			name:  "function application / invisible times",
			input: "X\u2061Y\u2062Z\u2063W",
			want:  "xyzw",
		},
		{name: "mongolian vowel separator", input: "M\u180Eongolia", want: "mongolia"},
		{name: "control chars", input: "Hello\x01\x02World\x03", want: "helloworld"},
		{name: "tabs, newlines, carriage returns", input: "A \tB\nC\rD", want: "a_b_c_d"},
		{
			name:  "mixed invisible + visible",
			input: "\u200BStart \uFEFFMiddle\u2060End",
			want:  "start_middleend",
		},
		{
			name:    "only invisible chars",
			input:   "\u200B\u200C\u200D\uFEFF\u2060\u2061\u2062\u2063\u180E",
			want:    "fallback",
			options: CanonifyOptions{Fallback: "fallback"},
		},

		// === Uniqueness / deterministic suffix ===
		{
			name:     "uniqueness deterministic",
			input:    "John",
			want:     "john_3",
			existing: []string{"john", "john_1", "john_2"},
		},
		{name: "exhaust attempts", input: "A", options: CanonifyOptions{MaxAttempts: 3},
			existing: []string{"a", "a_1", "a_2", "a_3"}, expectErr: true},
		{name: "nil exists func", input: "x", NotExistsFunc: true, expectErr: true},

		// === Complex normalization ===
		{
			name:  "ligatures and composed chars",
			input: "SŌME text—with–various―dashes and ﬁ ligature and café",
			want:  "some_text_with_various_dashes_and_fi_ligature_and_cafe",
		},

		// === Non-Latin scripts ===
		{name: "chinese", input: "中文 测试", want: "中文_测试"},
		{name: "cyrillic", input: "Русский-текст", want: "русскии_текст"},
		{name: "arabic", input: "مرحبا بالعالم", want: "مرحبا_بالعالم"},

		// === Emoji / symbols ===
		{name: "emoji", input: "emoji 😀 test", want: "emoji_test"},
		{name: "fancy unicode", input: "𝔉𝔞𝔫𝔠𝔶 𝕤𝕥𝕦𝕗𝕗", want: "fancy_stuff"},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			db := makeMock(tc.existing...)
			opts := tc.options
			if opts.Separator == 0 {
				opts.Separator = '_'
			}
			if opts.MinLen == 0 {
				opts.MinLen = 1
			}
			if opts.Fallback == "" {
				opts.Fallback = "item"
			}
			if opts.MaxAttempts == 0 {
				opts.MaxAttempts = 100
			}
			var ExistsFunc ExistsFunc
			if !tc.NotExistsFunc {
				ExistsFunc = db.Exists
			}
			got, err := CanonifyWithOptions(tc.input, ExistsFunc, opts)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			}
		})
	}
}

// Test cases are derived from the "Big List of Naughty Strings" repository:
// https://github.com/minimaxir/big-list-of-naughty-strings
func TestCanonifyBLNS(t *testing.T) {
	// Load BLNS corpus from testdata
	data, err := os.ReadFile("testdata/blns.json")
	require.NoError(t, err, "failed to read blns.json")

	var naughty []string
	require.NoError(t, json.Unmarshal(data, &naughty))

	// Load expected outputs
	expectedFile := "testdata/expected_blns.json"
	expData, err := os.ReadFile(expectedFile)
	require.NoError(t, err, "failed to read %s", expectedFile)

	expected := make(map[string]string)
	require.NoError(t, json.Unmarshal(expData, &expected))

	for i, s := range naughty {
		tcName := fmt.Sprintf("blns-%d", i)
		t.Run(tcName, func(t *testing.T) {
			db := makeMock()

			got, err := Canonify(s, db.Exists)
			require.NoError(t, err, "Canonify failed on input %q", s)

			exp, ok := expected[s]
			require.True(t, ok, "no expected output for input %q", s)
			require.Equal(t, exp, got, "output changed for input %q", s)
		})
	}
}
