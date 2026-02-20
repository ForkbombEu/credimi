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
)

// NormalizePipelineIdentifier trims whitespace and leading/trailing slashes from identifiers.
func NormalizePipelineIdentifier(identifier string) string {
	return strings.Trim(strings.TrimSpace(identifier), "/")
}

// PipelineTypedSearchAttributes returns typed search attributes for scheduled workflow actions.
func PipelineTypedSearchAttributes(pipelineIdentifier string) temporal.SearchAttributes {
	normalized := NormalizePipelineIdentifier(pipelineIdentifier)
	if normalized == "" {
		return temporal.NewSearchAttributes()
	}
	key := temporal.NewSearchAttributeKeyKeyword(PipelineIdentifierSearchAttribute)
	return temporal.NewSearchAttributes(key.ValueSet(normalized))
}

// ApplyPipelineSearchAttributes ensures StartWorkflowOptions include the pipeline identifier attribute.
func ApplyPipelineSearchAttributes(options *client.StartWorkflowOptions, pipelineIdentifier string) {
	if options == nil {
		return
	}
	normalized := NormalizePipelineIdentifier(pipelineIdentifier)
	if normalized == "" {
		return
	}

	key := temporal.NewSearchAttributeKeyKeyword(PipelineIdentifierSearchAttribute)
	if options.TypedSearchAttributes.Size() > 0 {
		options.TypedSearchAttributes = temporal.NewSearchAttributes(
			options.TypedSearchAttributes.Copy(),
			key.ValueSet(normalized),
		)
		options.SearchAttributes = nil
		return
	}
	options.TypedSearchAttributes = temporal.NewSearchAttributes(key.ValueSet(normalized))
	options.SearchAttributes = nil
}
