// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package reportpdf

import (
	"strings"
	"unicode"
)

// categoryMeta carries the display name and accent colour for one FCAF area.
// Colours mirror the webapp --fcaf-* tokens so the PDF and the web report stay
// visually consistent.
type categoryMeta struct {
	name  string
	color [3]int
}

// categoryOrder is the canonical display order, matching the webapp
// FCAF_CATEGORY_ORDER. It is authoritative because executed-test order alone
// would sort areas alphabetically (DM, IA, MS, …) rather than by FCAF flow.
var categoryOrder = []string{"DM", "MS", "IA", "SM", "SH", "UC", "OTHER"}

var categoryByCode = map[string]categoryMeta{
	"DM":    {"Data model", [3]int{78, 91, 210}},
	"MS":    {"Message structure", [3]int{14, 148, 160}},
	"IA":    {"Interaction", [3]int{201, 122, 20}},
	"SM":    {"Security mechanisms", [3]int{185, 49, 139}},
	"SH":    {"Shared", [3]int{31, 138, 92}},
	"UC":    {"Use cases", [3]int{142, 122, 18}},
	"OTHER": {"Other", [3]int{99, 99, 109}},
}

var subgroupLabels = map[string]string{
	"addressdata":        "Address data",
	"identifyingdata":    "Identifying data",
	"credentialmetadata": "Credential metadata",
	"protocolmessages":   "Protocol messages",
	"metadata":           "Metadata",
	"credentialformats":  "Credential formats",
	"maininteraction":    "Main interaction",
	"engagement":         "Engagement",
	"protocolflow":       "Protocol flow",
	"supportive":         "Supportive",
	"rpintegrity":        "RP integrity",
	"trustmechanisms":    "Trust mechanisms",
	"sessionencryption":  "Session encryption",
	"devicebinding":      "Device binding",
	"sessionbinding":     "Session binding",
	"issuerintegrity":    "Issuer integrity",
	"encoding":           "Encoding",
	"cryptography":       "Cryptography",
	"presentation":       "Presentation",
}

// parseTestID derives the FCAF area and subsection from a test identifier of
// the form WS_RP_<CATEGORY>_<SUBGROUP>_<specifics>[_|__]<NNN>. Test identifiers
// are the stable source of truth: the suite.section field mixes semantic paths
// with ARF clause references and cannot be relied upon.
func parseTestID(testID string) (code, subgroup, label string) {
	parts := strings.Split(testID, "_")
	if len(parts) >= 4 && parts[0] == "WS" && parts[1] == "RP" {
		code = strings.ToUpper(parts[2])
		if _, ok := categoryByCode[code]; ok {
			subgroup = strings.ToLower(parts[3])
			return code, subgroup, subgroupLabel(subgroup)
		}
	}
	return "OTHER", "other", "Other"
}

func subgroupLabel(key string) string {
	if label, ok := subgroupLabels[key]; ok {
		return label
	}
	return humanize(key)
}

// humanize turns a CamelCase segment into space-separated words, used only for
// subgroups not yet present in subgroupLabels.
func humanize(segment string) string {
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
