// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

// CredimiCapabilitiesMemoKey identifies Credimi-owned workflow capabilities in Temporal memo.
const CredimiCapabilitiesMemoKey = "credimi_capabilities"

// CredimiCapabilities describes optional product capabilities of one workflow execution.
type CredimiCapabilities struct {
	Logs bool `json:"logs"`
}

// WithCredimiCapabilities returns an input with authoritative capabilities merged into its memo.
func WithCredimiCapabilities(
	input WorkflowInput,
	capabilities CredimiCapabilities,
) WorkflowInput {
	input.Config = copyAnyMap(input.Config)
	memo, _ := input.Config["memo"].(map[string]any)
	memo = copyAnyMap(memo)
	memo[CredimiCapabilitiesMemoKey] = capabilities
	input.Config["memo"] = memo
	return input
}
