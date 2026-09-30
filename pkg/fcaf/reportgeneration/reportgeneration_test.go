// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package reportgeneration

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnrichReportJSONRejectsNonObjectRoots(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rawJSON string
	}{
		{name: "null", rawJSON: "null"},
		{name: "array", rawJSON: `[]`},
		{name: "string", rawJSON: `"report"`},
		{name: "number", rawJSON: `1`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			enriched, report, err := EnrichReportJSON(nil, nil, []byte(tt.rawJSON))
			require.Error(t, err)
			require.Nil(t, enriched)
			require.Nil(t, report)
			require.Contains(t, err.Error(), "decode FCAF report object")
		})
	}
}

func TestEnrichReportJSONAttachesPresentationToObjectRoot(t *testing.T) {
	t.Parallel()

	enriched, report, err := EnrichReportJSON(nil, nil, []byte(`{"status":"passed"}`))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, report.Presentation)
	require.Contains(t, string(enriched), `"presentation"`)
	require.Contains(t, string(enriched), `"status": "passed"`)
}
