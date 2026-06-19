// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalclient

import (
	"testing"

	temporalmocks "go.temporal.io/sdk/mocks"
)

func TestSetClientForTestsAndClearTestClients(t *testing.T) {
	ClearTestClients()
	if testClientCache != nil {
		t.Fatal("expected nil test cache")
	}

	mockClient := &temporalmocks.Client{}
	SetClientForTests("ns-1", mockClient)

	got, err := GetTemporalClientWithNamespace("ns-1")
	if err != nil {
		t.Fatalf("GetTemporalClientWithNamespace failed: %v", err)
	}
	if got != mockClient {
		t.Fatal("expected injected test client")
	}

	ClearTestClients()
	if testClientCache != nil {
		t.Fatal("expected nil test cache after clear")
	}
}
