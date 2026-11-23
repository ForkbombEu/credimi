// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipelineexpr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpressionRefs(t *testing.T) {
	require.Equal(
		t,
		[]string{"a.outputs.body.id", "b.outputs | optional"},
		ExpressionRefs("x ${{a.outputs.body.id }} y ${{ b.outputs | optional }}"),
	)
	require.Empty(t, ExpressionRefs("${fixture.issuer_url}"))
}

func TestFixtureDefaults(t *testing.T) {
	require.Equal(t, map[string]string{
		"issuer_url":   DefaultIssuerURL,
		"verifier_url": DefaultVerifierURL,
		"log_checker":  DefaultLogChecker,
	}, FixtureDefaults())
}
