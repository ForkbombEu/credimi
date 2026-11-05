// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package observability

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForceServiceName(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name: "forged service.name replaced, other attributes and large ints kept",
			body: `{"resourceSpans":[{"resource":{"attributes":[` +
				`{"key":"service.name","value":{"stringValue":"credimi"}},` +
				`{"key":"browser.platform","value":{"stringValue":"Linux"}}]},` +
				`"scopeSpans":[{"spans":[{"startTimeUnixNano":1793808000123456789}]}]}]}`,
		},
		{name: "resource missing", body: `{"resourceSpans":[{"scopeSpans":[]}]}`},
		{name: "not OTLP traces", body: `{"resourceLogs":[]}`, wantErr: true},
		{name: "not JSON", body: `garbage`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := forceServiceName([]byte(tt.body), BrowserServiceName)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			var payload struct {
				ResourceSpans []struct {
					Resource struct {
						Attributes []struct {
							Key   string         `json:"key"`
							Value map[string]any `json:"value"`
						} `json:"attributes"`
					} `json:"resource"`
					ScopeSpans []struct {
						Spans []map[string]json.Number `json:"spans"`
					} `json:"scopeSpans"`
				} `json:"resourceSpans"`
			}
			require.NoError(t, json.Unmarshal(out, &payload))
			services := map[string]int{}
			for _, rs := range payload.ResourceSpans {
				for _, attr := range rs.Resource.Attributes {
					if attr.Key == "service.name" {
						services[attr.Value["stringValue"].(string)]++
					}
				}
			}
			require.Equal(
				t,
				map[string]int{BrowserServiceName: len(payload.ResourceSpans)},
				services,
			)
			if tt.name[0] == 'f' {
				require.Len(t, payload.ResourceSpans[0].Resource.Attributes, 2)
				require.Equal(
					t,
					json.Number("1793808000123456789"),
					payload.ResourceSpans[0].ScopeSpans[0].Spans[0]["startTimeUnixNano"],
				)
			}
		})
	}
}
