// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package temporalclient provides functions to create and manage Temporal clients.
// It includes utilities for connecting to Temporal servers with default or custom namespaces.
package temporalclient

import (
	"fmt"
	"sync"

	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"go.temporal.io/sdk/client"
)

var (
	clientCache     sync.Map // map[string]client.Client under the hood
	testClientCache *sync.Map
	newLazyClient   = client.NewLazyClient
)

func getTemporalClient(args ...string) (client.Client, error) {
	namespace := "default"
	if len(args) > 0 {
		namespace = args[0]
	}
	if testClientCache != nil {
		if c, ok := testClientCache.Load(namespace); ok {
			return c.(client.Client), nil
		}
	}
	if c, ok := clientCache.Load(namespace); ok {
		return c.(client.Client), nil
	}
	options := baseOptions(namespace)
	options.DataConverter = temporalcrypto.DataConverter()
	c, err := newLazyClient(options)
	if err != nil {
		return nil, fmt.Errorf("unable to create client: %w", err)
	}

	if existing, loaded := clientCache.LoadOrStore(namespace, c); loaded {
		c.Close()
		return existing.(client.Client), nil
	}
	return c, nil
}

// GetTemporalClientWithNamespace returns the shared cached Temporal client for the namespace.
// Callers must not close the returned client; its lifecycle is managed by this package and
// ShutdownClients.
func GetTemporalClientWithNamespace(namespace string) (client.Client, error) {
	return getTemporalClient(namespace)
}

// NewNamespaceClient returns a new Temporal namespace client for TEMPORAL_ADDRESS.
// Callers own the returned client and must close it.
func NewNamespaceClient() (client.NamespaceClient, error) {
	return client.NewNamespaceClient(baseOptions(""))
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
