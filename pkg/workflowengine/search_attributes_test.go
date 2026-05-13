// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

func TestNormalizePipelineIdentifier(t *testing.T) {
	require.Equal(t, "tenant/pipeline", NormalizePipelineIdentifier(" /tenant/pipeline/ "))
	require.Empty(t, NormalizePipelineIdentifier(" / "))
}

func TestPipelineTypedSearchAttributes(t *testing.T) {
	attrs := PipelineTypedSearchAttributes(
		" /tenant/pipeline/ ",
		[]string{"runner-1"},
		EntityIDs{
			Actions:           []string{"action-1"},
			Versions:          []string{"version-1"},
			Credentials:       []string{"credential-1"},
			UseCases:          []string{"use-case-1"},
			ConformanceChecks: []string{"conformance-1"},
			CustomChecks:      []string{"custom-1"},
		},
	)

	pipelineKey := temporal.NewSearchAttributeKeyKeyword(PipelineIdentifierSearchAttribute)
	pipelineIdentifier, ok := attrs.GetKeyword(pipelineKey)
	require.True(t, ok)
	require.Equal(t, "tenant/pipeline", pipelineIdentifier)

	runnerKey := temporal.NewSearchAttributeKeyKeywordList(RunnerIdentifiersSearchAttribute)
	runnerIDs, ok := attrs.GetKeywordList(runnerKey)
	require.True(t, ok)
	require.Equal(t, []string{"runner-1"}, runnerIDs)

	actionKey := temporal.NewSearchAttributeKeyKeywordList(ActionsSearchAttribute)
	actionIDs, ok := attrs.GetKeywordList(actionKey)
	require.True(t, ok)
	require.Equal(t, []string{"action-1"}, actionIDs)
}

func TestApplyPipelineSearchAttributes(t *testing.T) {
	options := client.StartWorkflowOptions{
		TypedSearchAttributes: PipelineTypedSearchAttributes(
			"existing/pipeline",
			nil,
			EntityIDs{},
		),
	}

	ApplyPipelineSearchAttributes(
		&options,
		"tenant/pipeline",
		[]string{"runner-1"},
		EntityIDs{
			Actions:           []string{"action-1"},
			Versions:          []string{"version-1"},
			Credentials:       []string{"credential-1"},
			UseCases:          []string{"use-case-1"},
			ConformanceChecks: []string{"conformance-1"},
			CustomChecks:      []string{"custom-1"},
		},
	)

	pipelineKey := temporal.NewSearchAttributeKeyKeyword(PipelineIdentifierSearchAttribute)
	pipelineIdentifier, ok := options.TypedSearchAttributes.GetKeyword(pipelineKey)
	require.True(t, ok)
	require.Equal(t, "tenant/pipeline", pipelineIdentifier)

	credentialKey := temporal.NewSearchAttributeKeyKeywordList(CredentialsSearchAttribute)
	credentialIDs, ok := options.TypedSearchAttributes.GetKeywordList(credentialKey)
	require.True(t, ok)
	require.Equal(t, []string{"credential-1"}, credentialIDs)

	customCheckKey := temporal.NewSearchAttributeKeyKeywordList(CustomCheckSearchAttribute)
	customCheckIDs, ok := options.TypedSearchAttributes.GetKeywordList(customCheckKey)
	require.True(t, ok)
	require.Equal(t, []string{"custom-1"}, customCheckIDs)
}

func TestApplyPipelineSearchAttributesSkipsEmptyInput(t *testing.T) {
	ApplyPipelineSearchAttributes(nil, "tenant/pipeline", nil, EntityIDs{})

	options := client.StartWorkflowOptions{}
	ApplyPipelineSearchAttributes(&options, " / ", nil, EntityIDs{})

	require.Equal(t, 0, options.TypedSearchAttributes.Size())
}
