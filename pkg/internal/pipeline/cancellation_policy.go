// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

const PipelineCancellationPolicySignal = "pipeline-cancellation-policy"

type PipelineCancellationPolicy struct {
	Reason               string   `json:"reason"`
	SkipRunnerCleanup    bool     `json:"skip_runner_cleanup"`
	SkipRunnerCleanupIDs []string `json:"skip_runner_cleanup_ids,omitempty"`
}
