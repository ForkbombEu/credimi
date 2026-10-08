// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalclient

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		require.NotNil(t, options.DataConverter)
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

func TestGetTemporalClientWithNamespaceConcurrentFirstUse(t *testing.T) {
	ShutdownClients()

	origNewLazy := newLazyClient
	t.Cleanup(func() {
		newLazyClient = origNewLazy
		ShutdownClients()
	})

	var closed atomic.Int32
	var closedClient atomic.Pointer[temporalmocks.Client]
	mocksByCall := []*temporalmocks.Client{{}, {}}
	for _, m := range mocksByCall {
		m.On("Close").Run(func(mock.Arguments) {
			closed.Add(1)
			closedClient.Store(m)
		}).Return().Maybe()
	}

	// Both callers miss the cache and create a client before either stores it.
	var calls atomic.Int32
	bothCreating := make(chan struct{})
	newLazyClient = func(_ client.Options) (client.Client, error) {
		n := calls.Add(1)
		if n == 2 {
			close(bothCreating)
		}
		select {
		case <-bothCreating:
		case <-time.After(time.Second):
		}
		return mocksByCall[n-1], nil
	}

	results := make([]client.Client, 2)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := GetTemporalClientWithNamespace("x")
			assert.NoError(t, err)
			results[i] = c
		}()
	}
	wg.Wait()

	require.Equal(t, int32(2), calls.Load())
	require.Same(t, results[0], results[1])
	require.Equal(t, int32(1), closed.Load())
	require.NotSame(t, results[0], closedClient.Load())
	cached, ok := clientCache.Load("x")
	require.True(t, ok)
	require.Same(t, results[0], cached)
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

func TestGetTemporalClientWithNamespaceReturnsCreateError(t *testing.T) {
	ShutdownClients()

	origNewLazy := newLazyClient
	t.Cleanup(func() {
		newLazyClient = origNewLazy
		ShutdownClients()
	})

	newLazyClient = func(options client.Options) (client.Client, error) {
		require.NotNil(t, options.DataConverter)
		return nil, assertError("boom")
	}

	_, err := GetTemporalClientWithNamespace("default")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to create client")
}

func assertError(message string) error {
	return &testError{message: message}
}

type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}
