// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func TestParseFidesCredentialIssuersActivity(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestActivityEnvironment()
	act := NewParseFidesCredentialIssuersActivity()
	env.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{Name: act.Name()})

	input := workflowengine.ActivityInput{
		Payload: ParseFidesCredentialIssuersActivityPayload{
			Data: map[string]any{
				"content": []any{
					map[string]any{
						"issuanceProtocol":   "oid4vci",
						"oid4vciMetadataUrl": "https://example.com/.well-known/openid-credential-issuer/issuer-1",
					},
					map[string]any{
						"issuanceProtocol":    "oid4vci",
						"credentialIssuerUrl": "https://example.com/issuer-2",
					},
					map[string]any{
						"issuanceProtocol":    "other",
						"credentialIssuerUrl": "https://example.com/non-oid4vci",
					},
				},
				"number":     float64(0),
				"totalPages": float64(1),
			},
		},
	}

	future, err := env.ExecuteActivity(act.Name(), input)
	require.NoError(t, err)

	var result workflowengine.ActivityResult
	require.NoError(t, future.Get(&result))
	output, ok := result.Output.(map[string]any)
	require.True(t, ok)
	require.Equal(
		t,
		[]any{
			"https://example.com/issuer-1",
			"https://example.com/issuer-2",
		},
		output["issuers"],
	)
	require.Equal(t, float64(0), output["page_number"])
	require.Equal(t, float64(1), output["total_pages"])
}

func TestParseFidesCredentialIssuersActivityInvalidInput(t *testing.T) {
	act := NewParseFidesCredentialIssuersActivity()
	_, err := act.Execute(context.TODO(), workflowengine.ActivityInput{
		Payload: ParseFidesCredentialIssuersActivityPayload{
			Data: func() {},
		},
	})
	require.Error(t, err)
}
