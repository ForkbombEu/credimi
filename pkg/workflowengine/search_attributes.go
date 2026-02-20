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

// PipelineSearchAttributes returns search attributes for the provided pipeline identifier.
func PipelineSearchAttributes(pipelineIdentifier string) map[string]any {
	normalized := NormalizePipelineIdentifier(pipelineIdentifier)
	if normalized == "" {
		return nil
	}
	return map[string]any{PipelineIdentifierSearchAttribute: normalized}
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
	attributes := PipelineSearchAttributes(pipelineIdentifier)
	if len(attributes) == 0 {
		return
	}
	if options.SearchAttributes == nil {
		options.SearchAttributes = map[string]any{}
	}
	for key, value := range attributes {
		options.SearchAttributes[key] = value
	}
}
