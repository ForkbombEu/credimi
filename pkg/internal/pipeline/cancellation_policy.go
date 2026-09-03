// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

const PipelineCancellationPolicySignal = "pipeline-cancellation-policy"

type PipelineCancellationPolicy struct {
	Reason               string   `json:"reason"`
	SkipDeviceCleanup    bool     `json:"skip_device_cleanup"`
	SkipDeviceCleanupIDs []string `json:"skip_device_cleanup_ids,omitempty"`
}
