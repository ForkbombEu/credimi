// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package reportpdf

import "github.com/forkbombeu/credimi/pkg/fcaf/taxonomy"

// categoryMeta carries the PDF accent colour for one FCAF area.
// Colours mirror the webapp --fcaf-* tokens so the PDF and the web report stay
// visually consistent. Labels and order come from pkg/fcaf/taxonomy.
type categoryMeta struct {
	color [3]int
}

var categoryColors = map[string]categoryMeta{
	"DM":    {[3]int{78, 91, 210}},
	"MS":    {[3]int{14, 148, 160}},
	"IA":    {[3]int{201, 122, 20}},
	"SM":    {[3]int{185, 49, 139}},
	"SH":    {[3]int{31, 138, 92}},
	"UC":    {[3]int{142, 122, 18}},
	"OTHER": {[3]int{99, 99, 109}},
}

func parseTestID(testID string) (code, subgroup, label string) {
	parsed := taxonomy.Parse(testID)
	return parsed.Code, parsed.Subgroup, parsed.Label
}

func categoryName(code string) string {
	return taxonomy.LabelFor(code)
}

func categoryColor(code string) [3]int {
	if meta, ok := categoryColors[code]; ok {
		return meta.color
	}
	return categoryColors["OTHER"].color
}
