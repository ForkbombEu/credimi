// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

// CredimiCapabilitiesMemoKey identifies Credimi-owned workflow capabilities in Temporal memo.
const CredimiCapabilitiesMemoKey = "credimi_capabilities"

// LinkedStepsMemoKey identifies the memo of a Linked Steps Workflow: its step IDs and the
// attempts its retry policy allows.
const LinkedStepsMemoKey = "credimi_linked_steps"

// LinkedStepsMemo is the memo of a Linked Steps Workflow run.
type LinkedStepsMemo struct {
	Steps           []string `json:"steps"`
	MaximumAttempts int32    `json:"maximum_attempts"`
}

// CredimiCapabilities describes optional product capabilities of one workflow execution.
type CredimiCapabilities struct {
	Logs bool `json:"logs"`
	QR   bool `json:"qr"`
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
