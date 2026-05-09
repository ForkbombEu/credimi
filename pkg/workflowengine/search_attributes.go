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
	ActionsSearchAttribute           = "ActionsID"
	VersionsSearchAttribute          = "VersionsID"
	CredentialsSearchAttribute       = "CredentialsID"
	UseCaseSearchAttribute           = "UseCaseID"
	ConformanceCheckSearchAttribute  = "ConformanceCheckID"
	CustomCheckSearchAttribute       = "CustomCheckID"
)

type EntityIDs struct {
    Actions           []string `json:"actions,omitempty"`
    Versions          []string `json:"versions,omitempty"`
    Credentials       []string `json:"credentials,omitempty"`
    UseCases          []string `json:"use_cases,omitempty"`
    ConformanceChecks []string `json:"conformance_checks,omitempty"`
    CustomChecks      []string `json:"custom_checks,omitempty"`
}

// NormalizePipelineIdentifier trims whitespace and leading/trailing slashes from identifiers.
func NormalizePipelineIdentifier(identifier string) string {
	return strings.Trim(strings.TrimSpace(identifier), "/")
}

// PipelineTypedSearchAttributes returns typed search attributes for scheduled workflow actions.
func PipelineTypedSearchAttributes(
	pipelineIdentifier string, 
	runnerIDs []string,
	entityIDs EntityIDs,
) temporal.SearchAttributes {
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
	if len(entityIDs.Actions) > 0 {
		key := temporal.NewSearchAttributeKeyKeywordList(ActionsSearchAttribute)
		attrs = append(attrs, key.ValueSet(entityIDs.Actions))
	}
	if len(entityIDs.Versions) > 0 {
		key := temporal.NewSearchAttributeKeyKeywordList(VersionsSearchAttribute)
		attrs = append(attrs, key.ValueSet(entityIDs.Versions))
	}
	if len(entityIDs.Credentials) > 0 {
		key := temporal.NewSearchAttributeKeyKeywordList(CredentialsSearchAttribute)
		attrs = append(attrs, key.ValueSet(entityIDs.Credentials))
	}
	if len(entityIDs.UseCases) > 0 {
		key := temporal.NewSearchAttributeKeyKeywordList(UseCaseSearchAttribute)
		attrs = append(attrs, key.ValueSet(entityIDs.UseCases))
	}
	if len(entityIDs.ConformanceChecks) > 0 {
		key := temporal.NewSearchAttributeKeyKeywordList(ConformanceCheckSearchAttribute)
		attrs = append(attrs, key.ValueSet(entityIDs.ConformanceChecks))
	}
	if len(entityIDs.CustomChecks) > 0 {
		key := temporal.NewSearchAttributeKeyKeywordList(CustomCheckSearchAttribute)
		attrs = append(attrs, key.ValueSet(entityIDs.CustomChecks))
	}
	if len(attrs) == 0 {
		return temporal.NewSearchAttributes()
	}
	return temporal.NewSearchAttributes(attrs...)
}

// ApplyPipelineSearchAttributes ensures StartWorkflowOptions include pipeline visibility attributes.
func ApplyPipelineSearchAttributes(
	options *client.StartWorkflowOptions,
	pipelineIdentifier string,
	runnerIDs []string,
	entityIDs EntityIDs,
) {
	if options == nil {
		return
	}

	updates := []temporal.SearchAttributeUpdate{}
	if normalized := NormalizePipelineIdentifier(pipelineIdentifier); normalized != "" {
		pipelineKey := temporal.NewSearchAttributeKeyKeyword(PipelineIdentifierSearchAttribute)
		updates = append(updates, pipelineKey.ValueSet(normalized))
	}
	if len(runnerIDs) > 0 {
		runnerKey := temporal.NewSearchAttributeKeyKeywordList(RunnerIdentifiersSearchAttribute)
		updates = append(updates, runnerKey.ValueSet(runnerIDs))
	}
	if len(entityIDs.Actions) > 0 {
		key := temporal.NewSearchAttributeKeyKeywordList(ActionsSearchAttribute)
		updates = append(updates, key.ValueSet(entityIDs.Actions))
	}
	if len(entityIDs.Versions) > 0 {
		key := temporal.NewSearchAttributeKeyKeywordList(VersionsSearchAttribute)
		updates = append(updates, key.ValueSet(entityIDs.Versions))
	}
	if len(entityIDs.Credentials) > 0 {
		key := temporal.NewSearchAttributeKeyKeywordList(CredentialsSearchAttribute)
		updates = append(updates, key.ValueSet(entityIDs.Credentials))
	}
	if len(entityIDs.UseCases) > 0 {
		key := temporal.NewSearchAttributeKeyKeywordList(UseCaseSearchAttribute)
		updates = append(updates, key.ValueSet(entityIDs.UseCases))
	}
	if len(entityIDs.ConformanceChecks) > 0 {
		key := temporal.NewSearchAttributeKeyKeywordList(ConformanceCheckSearchAttribute)
		updates = append(updates, key.ValueSet(entityIDs.ConformanceChecks))
	}
	if len(entityIDs.CustomChecks) > 0 {
		key := temporal.NewSearchAttributeKeyKeywordList(CustomCheckSearchAttribute)
		updates = append(updates, key.ValueSet(entityIDs.CustomChecks))
	}
	if len(updates) == 0 {
		return
	}
	if options.TypedSearchAttributes.Size() > 0 {
		updates = append([]temporal.SearchAttributeUpdate{options.TypedSearchAttributes.Copy()}, updates...)
	}
	options.TypedSearchAttributes = temporal.NewSearchAttributes(updates...)
}
