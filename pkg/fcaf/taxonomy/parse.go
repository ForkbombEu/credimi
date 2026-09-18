// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package taxonomy is the shared FCAF category/subgroup vocabulary.
package taxonomy

import (
	"strings"
	"unicode"
)

// Category is one FCAF area (DM, MS, IA, …).
type Category struct {
	Code  string
	Label string
}

// Parsed is the category and subgroup derived from a test identifier.
type Parsed struct {
	Code     string
	Subgroup string
	Label    string
}

// Parse derives the FCAF area and subsection from a test identifier of the
// form WS_RP_<CATEGORY>_<SUBGROUP>_<specifics>[_|__]<NNN>.
func Parse(testID string) Parsed {
	if testID == "" {
		return other()
	}
	parts := strings.Split(testID, "_")
	if len(parts) >= 4 && parts[0] == "WS" && parts[1] == "RP" {
		code := strings.ToUpper(parts[2])
		if _, ok := Categories[code]; ok {
			segment := parts[3]
			return Parsed{
				Code:     code,
				Subgroup: strings.ToLower(segment),
				Label:    SubgroupLabel(segment),
			}
		}
	}
	return other()
}

func other() Parsed {
	return Parsed{Code: "OTHER", Subgroup: "other", Label: "Other"}
}

// SubgroupLabel returns the display label for a subgroup segment. Lookup is
// case-insensitive; unknown segments are humanized from their original casing.
func SubgroupLabel(segment string) string {
	key := strings.ToLower(segment)
	if label, ok := SubgroupLabels[key]; ok {
		return label
	}
	return Humanize(segment)
}

// Humanize turns a CamelCase segment into space-separated words.
func Humanize(segment string) string {
	if segment == "" {
		return ""
	}
	var sb strings.Builder
	runes := []rune(segment)
	sb.WriteRune(unicode.ToUpper(runes[0]))
	for i := 1; i < len(runes); i++ {
		if unicode.IsUpper(runes[i]) {
			sb.WriteByte(' ')
		}
		sb.WriteRune(runes[i])
	}
	return sb.String()
}

// LabelFor returns the display label for a category code, or "Other".
func LabelFor(code string) string {
	if cat, ok := Categories[code]; ok {
		return cat.Label
	}
	return Categories["OTHER"].Label
}
