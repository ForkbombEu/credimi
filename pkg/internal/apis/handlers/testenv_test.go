// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import "os"

func init() {
	_ = os.Setenv(
		"CREDIMI_TEMPORAL_SECRETS_ENCRYPTION_KEY",
		"MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDA=",
	)
}
