// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package temporalclient provides functions to create and manage Temporal clients.
// It includes utilities for connecting to Temporal servers with default or custom namespaces.
package temporalclient

import (
	"fmt"
	"sync"

	"github.com/forkbombeu/credimi/internal/telemetry"
	"github.com/forkbombeu/credimi/pkg/utils"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

var (
	clientCache sync.Map // map[string]client.Client under the hood
)

func getTemporalClient(args ...string) (client.Client, error) {
	namespace := "default"
	if len(args) > 0 {
		namespace = args[0]
	}
	if c, ok := clientCache.Load(namespace); ok {
		return c.(client.Client), nil
	}
	hostPort := utils.GetEnvironmentVariable("TEMPORAL_ADDRESS", client.DefaultHostPort)
	contextPropagators := []workflow.ContextPropagator{
		telemetry.NewTraceContextPropagator(),
	}
	c, err := client.NewLazyClient(client.Options{
		HostPort:           hostPort,
		Namespace:          namespace,
		ContextPropagators: contextPropagators,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create client: %w", err)
	}

	clientCache.Store(namespace, c)
	return c, nil
}

// GetTemporalClientWithNamespace creates a new Temporal client with the specified namespace.
// It uses the TEMPORAL_ADDRESS environment variable to determine the host and port.
// If TEMPORAL_ADDRESS is not set, it defaults to client.DefaultHostPort.
func GetTemporalClientWithNamespace(namespace string) (client.Client, error) {
	return getTemporalClient(namespace)
}

// GetTemporalDebugClient creates a Temporal client with masked payload decoding for debugging.
func GetTemporalDebugClient(namespace string) (client.Client, error) {
	hostPort := utils.GetEnvironmentVariable("TEMPORAL_ADDRESS", client.DefaultHostPort)
	contextPropagators := []workflow.ContextPropagator{
		telemetry.NewTraceContextPropagator(),
	}
	return client.NewLazyClient(client.Options{
		HostPort:           hostPort,
		Namespace:          namespace,
		ContextPropagators: contextPropagators,
		DataConverter:      NewPrettyMaskingDataConverter(),
	})
}

func ShutdownClients() {
	clientCache.Range(func(key, value any) bool {
		if c, ok := value.(client.Client); ok {
			c.Close()
			clientCache.Delete(key)
		}
		return true
	})
}
