// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalclient

import (
	internal "github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"go.temporal.io/sdk/client"
)

// GetTemporalClientWithNamespace forwards to the internal Temporal client factory.
func GetTemporalClientWithNamespace(namespace string) (client.Client, error) {
	return internal.GetTemporalClientWithNamespace(namespace)
}

// GetTemporalDebugClient forwards to the internal debug client factory.
func GetTemporalDebugClient(namespace string) (client.Client, error) {
	return internal.GetTemporalDebugClient(namespace)
}
