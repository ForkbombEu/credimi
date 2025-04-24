// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalclient

import (
	"fmt"

	"github.com/forkbombeu/didimo/pkg/utils"
	"go.temporal.io/sdk/client"
)

func getTemporalClient(args ...string) (client.Client, error) {
	namespace := "default"
	if len(args) > 0 {
		namespace = args[0]
	}
	hostPort := utils.GetEnvironmentVariable("TEMPORAL_ADDRESS", client.DefaultHostPort)
	c, err := client.Dial(client.Options{
		HostPort:  hostPort,
		Namespace: namespace,
	})

	if err != nil {
		return nil, fmt.Errorf("unable to create client: %v", err)
	}
	return c, nil
}

// GetTemporalClient creates a new Temporal client with the default namespace.
// It uses the TEMPORAL_ADDRESS environment variable to determine the host and port.
// If TEMPORAL_ADDRESS is not set, it defaults to client.DefaultHostPort.
func GetTemporalClient() (client.Client, error) {
	c, err := getTemporalClient()
	if err != nil {
		return nil, fmt.Errorf("unable to create client: %v", err)
	}

	return c, nil
}

// GetTemporalClientWithNamespace creates a new Temporal client with the specified namespace.
// It uses the TEMPORAL_ADDRESS environment variable to determine the host and port.
// If TEMPORAL_ADDRESS is not set, it defaults to client.DefaultHostPort.
func GetTemporalClientWithNamespace(namespace string) (client.Client, error) {
	c, err := getTemporalClient(namespace)
	if err != nil {
		return nil, fmt.Errorf("unable to create client: %v", err)
	}

	return c, nil
}
