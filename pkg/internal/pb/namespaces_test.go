// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
)

type fakeNamespaceClient struct {
	describeErrs []error
	describeCalls int
}

func (f *fakeNamespaceClient) Register(_ context.Context, _ *workflowservice.RegisterNamespaceRequest) error {
	return nil
}

func (f *fakeNamespaceClient) Describe(_ context.Context, _ string) (*workflowservice.DescribeNamespaceResponse, error) {
	f.describeCalls++
	if len(f.describeErrs) > 0 {
		err := f.describeErrs[0]
		f.describeErrs = f.describeErrs[1:]
		return nil, err
	}
	return &workflowservice.DescribeNamespaceResponse{}, nil
}

func (f *fakeNamespaceClient) Update(_ context.Context, _ *workflowservice.UpdateNamespaceRequest) error {
	return nil
}

func (f *fakeNamespaceClient) Close() {}

func TestWaitForNamespaceReadyImmediateSuccess(t *testing.T) {
	client := &fakeNamespaceClient{}

	err := waitForNamespaceReady(client, "default", time.Second)
	require.NoError(t, err)
	require.Equal(t, 1, client.describeCalls)
}

func TestWaitForNamespaceReadyRetriesThenSucceeds(t *testing.T) {
	client := &fakeNamespaceClient{describeErrs: []error{errors.New("transient")}}

	err := waitForNamespaceReady(client, "default", 3*time.Second)
	require.NoError(t, err)
	require.GreaterOrEqual(t, client.describeCalls, 2)
}

func TestWaitForNamespaceReadyTimeout(t *testing.T) {
	client := &fakeNamespaceClient{describeErrs: []error{errors.New("still failing")}}

	err := waitForNamespaceReady(client, "default", -time.Second)
	require.Error(t, err)
	require.Equal(t, 1, client.describeCalls)
}
