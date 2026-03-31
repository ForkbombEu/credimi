// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflowengine

import (
	"strings"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

const (
	// PipelineIdentifierSearchAttribute is the Temporal visibility field storing pipeline identifiers.
	PipelineIdentifierSearchAttribute = "PipelineIdentifier"
	// RunnerIdentifierSearchAttribute is the Temporal visibility field storing runner identifiers.
	RunnerIdentifiersSearchAttribute = "RunnerIdentifiers"
	ActionsSearchAttribute = "ActionsID"
    VersionsSearchAttribute = "VersionsID"
    CredentialsSearchAttribute = "CredentialsID"
    UseCaseSearchAttribute = "UseCaseID"
    ConformanceCheckSearchAttribute = "ConformanceCheckID"
    CustomCheckSearchAttribute = "CustomCheckID"
)

// NormalizePipelineIdentifier trims whitespace and leading/trailing slashes from identifiers.
func NormalizePipelineIdentifier(identifier string) string {
	return strings.Trim(strings.TrimSpace(identifier), "/")
}

// PipelineTypedSearchAttributes returns typed search attributes for scheduled workflow actions.
func PipelineTypedSearchAttributes(pipelineIdentifier string, runnerIDs []string) temporal.SearchAttributes {
	var attrs []temporal.SearchAttributeUpdate
	normalized := NormalizePipelineIdentifier(pipelineIdentifier)
	if normalized != "" {
		keyPipeline := temporal.NewSearchAttributeKeyKeyword(PipelineIdentifierSearchAttribute)
		attrs = append(attrs, keyPipeline.ValueSet(normalized))
	}
	if len(runnerIDs) > 0 {
		runnerKey := temporal.NewSearchAttributeKeyKeywordList(RunnerIdentifiersSearchAttribute)
		attrs = append(attrs, runnerKey.ValueSet(runnerIDs))
	}
	if len(attrs) == 0 {
		return temporal.NewSearchAttributes()
	}
	return temporal.NewSearchAttributes(attrs...)
}

// ApplyPipelineSearchAttributes ensures StartWorkflowOptions include the pipeline identifier attribute.
func ApplyPipelineSearchAttributes(options *client.StartWorkflowOptions, 
	pipelineIdentifier string,
	runnerIDs []string) {
	if options == nil {
		return
	}
	normalized := NormalizePipelineIdentifier(pipelineIdentifier)
	if normalized == "" {
		return
	}

	pipelineKey := temporal.NewSearchAttributeKeyKeyword(PipelineIdentifierSearchAttribute)
		if options.TypedSearchAttributes.Size() > 0 {
			options.TypedSearchAttributes = temporal.NewSearchAttributes(
				options.TypedSearchAttributes.Copy(),
				pipelineKey.ValueSet(normalized),
			)
			return
		}
		options.TypedSearchAttributes = temporal.NewSearchAttributes(pipelineKey.ValueSet(normalized))

	if len(runnerIDs) > 0 {
		runnerKey := temporal.NewSearchAttributeKeyKeywordList(RunnerIdentifiersSearchAttribute)
		if options.TypedSearchAttributes.Size() > 0 {
			options.TypedSearchAttributes = temporal.NewSearchAttributes(
				options.TypedSearchAttributes.Copy(),
				runnerKey.ValueSet(runnerIDs),
			)
			return
		}
		options.TypedSearchAttributes = temporal.NewSearchAttributes(runnerKey.ValueSet(runnerIDs))
	}
}
