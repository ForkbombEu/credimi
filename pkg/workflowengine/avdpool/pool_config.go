// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package avdpool

import (
	"strconv"

	"github.com/forkbombeu/credimi/pkg/utils"
)

func ConfigFromEnv() PoolConfig {
	config := DefaultPoolConfig()

	if maxStr := utils.GetEnvironmentVariable("AVD_POOL_MAX_CONCURRENT", ""); maxStr != "" {
		if max, err := strconv.Atoi(maxStr); err == nil {
			config.MaxConcurrentEmulators = max
		}
	}

	if queueStr := utils.GetEnvironmentVariable("AVD_POOL_MAX_QUEUE", ""); queueStr != "" {
		if queue, err := strconv.Atoi(queueStr); err == nil {
			config.MaxQueueDepth = queue
		}
	}

	return ApplyPoolConfigDefaults(config)
}
