// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPMDocDeviceResponsePerQueryValidator(t *testing.T) {
	numbers := []string{"CREDIMI-DEMO-001", "CREDIMI-DEMO-002", "CREDIMI-DEMO-U18"}
	ids := []string{"pid_a", "pid_b", "pid_c"}
	deviceResponse := func(number string) string {
		return encodedMSOStatusDeviceResponse(t, nil, map[string]any{"document_number": number})
	}
	exchange := func(vpToken map[string]any) map[string]any {
		credentials := make([]any, 0, len(ids))
		for index, id := range ids {
			credentials = append(credentials, map[string]any{
				"id":     id,
				"format": "mso_mdoc",
				"meta":   map[string]any{"doctype_value": msoStatusTestDocType},
				"claims": []any{map[string]any{
					"path":   []any{msoStatusTestDocType, "document_number"},
					"values": []any{numbers[index]},
				}},
			})
		}
		return map[string]any{
			"authorization_request": map[string]any{
				"dcql_query": map[string]any{"credentials": credentials},
			},
			"observed": map[string]any{"wallet_response": map[string]any{
				"value": map[string]any{"vp_token": vpToken},
			}},
		}
	}
	conformant := map[string]any{
		"pid_a": []any{deviceResponse(numbers[0])},
		"pid_b": []any{deviceResponse(numbers[1])},
		"pid_c": []any{deviceResponse(numbers[2])},
	}

	for _, test := range []struct {
		name    string
		vpToken map[string]any
		want    Status
	}{
		{"one matching DeviceResponse per query", conformant, StatusPass},
		{
			"one credential answering every query",
			map[string]any{
				"pid_a": []any{deviceResponse(numbers[0])},
				"pid_b": []any{deviceResponse(numbers[0])},
				"pid_c": []any{deviceResponse(numbers[0])},
			},
			StatusFail,
		},
		{
			"a query left unanswered",
			map[string]any{
				"pid_a": []any{deviceResponse(numbers[0])},
				"pid_b": []any{deviceResponse(numbers[1])},
			},
			StatusFail,
		},
		{
			"two DeviceResponses for one query",
			map[string]any{
				"pid_a": []any{deviceResponse(numbers[0]), deviceResponse(numbers[0])},
				"pid_b": []any{deviceResponse(numbers[1])},
				"pid_c": []any{deviceResponse(numbers[2])},
			},
			StatusFail,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := OID4VPMDocDeviceResponsePerQueryValidator{}.Validate(
				context.Background(),
				Input{
					Value:  exchange(test.vpToken),
					Params: map[string]any{"doctype": msoStatusTestDocType, "min_queries": 3},
				},
			)
			require.Equal(t, test.want, result.Status, result.Message)
		})
	}
}
