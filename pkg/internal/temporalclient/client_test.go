// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalclient

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func TestGetTemporalClientWithNamespaceCaches(t *testing.T) {
	ShutdownClients()

	origNewLazy := newLazyClient
	t.Cleanup(func() {
		newLazyClient = origNewLazy
		ShutdownClients()
	})

	mockDefault := &temporalmocks.Client{}
	mockOther := &temporalmocks.Client{}
	mockDefault.On("Close").Return(nil).Maybe()
	mockOther.On("Close").Return(nil).Maybe()

	callCount := 0
	newLazyClient = func(options client.Options) (client.Client, error) {
		callCount++
		if options.Namespace == "other" {
			return mockOther, nil
		}
		return mockDefault, nil
	}

	c1, err := GetTemporalClientWithNamespace("default")
	require.NoError(t, err)
	c2, err := GetTemporalClientWithNamespace("default")
	require.NoError(t, err)
	require.Same(t, c1, c2)

	c3, err := GetTemporalClientWithNamespace("other")
	require.NoError(t, err)
	require.Same(t, mockOther, c3)

	require.Equal(t, 2, callCount)
}

func TestShutdownClientsClearsCache(t *testing.T) {
	ShutdownClients()

	origNewLazy := newLazyClient
	t.Cleanup(func() {
		newLazyClient = origNewLazy
		ShutdownClients()
	})

	created := 0
	clients := []*temporalmocks.Client{}
	newLazyClient = func(options client.Options) (client.Client, error) {
		created++
		mockClient := &temporalmocks.Client{}
		mockClient.On("Close").Return(nil).Once()
		clients = append(clients, mockClient)
		return mockClient, nil
	}

	c1, err := GetTemporalClientWithNamespace("default")
	require.NoError(t, err)
	require.Same(t, clients[0], c1)

	ShutdownClients()

	c2, err := GetTemporalClientWithNamespace("default")
	require.NoError(t, err)
	require.NotSame(t, c1, c2)
	require.Equal(t, 2, created)

	ShutdownClients()
	for _, mockClient := range clients {
		mockClient.AssertExpectations(t)
	}
}
