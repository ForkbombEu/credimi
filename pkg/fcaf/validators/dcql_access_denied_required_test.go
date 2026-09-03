// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later
package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccessDeniedRequiredMode(t *testing.T) {
	validator := DCQLResponseConstraintsValidator{}
	query := map[string]any{
		"dcql_query": map[string]any{"credentials": []any{map[string]any{"id": "pid"}}},
	}
	withError := map[string]any{"dcql_query": query["dcql_query"], "error": "access_denied"}
	empty := query

	require.Equal(
		t,
		StatusPass,
		validator.Validate(
			context.Background(),
			Input{Value: withError, Params: map[string]any{"mode": "access_denied_required"}},
		).Status,
	)
	require.Equal(
		t,
		StatusFail,
		validator.Validate(
			context.Background(),
			Input{Value: empty, Params: map[string]any{"mode": "access_denied_required"}},
		).Status,
	)
}

func TestTransactionDataErrorRequiredMode(t *testing.T) {
	validator := DCQLResponseConstraintsValidator{}
	query := map[string]any{
		"dcql_query": map[string]any{"credentials": []any{map[string]any{"id": "pid"}}},
	}
	withError := map[string]any{
		"dcql_query": query["dcql_query"],
		"error":      "invalid_transaction_data",
	}

	require.Equal(
		t,
		StatusPass,
		validator.Validate(
			context.Background(),
			Input{
				Value:  withError,
				Params: map[string]any{"mode": "transaction_data_error_required"},
			},
		).Status,
	)
	require.Equal(
		t,
		StatusFail,
		validator.Validate(
			context.Background(),
			Input{Value: query, Params: map[string]any{"mode": "transaction_data_error_required"}},
		).Status,
	)
}
