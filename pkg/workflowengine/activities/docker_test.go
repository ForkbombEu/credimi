// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestDockerRunActivity_Execute(t *testing.T) {
	activity := &DockerActivity{}
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()
	env.RegisterActivity(activity.Execute)

	tests := []struct {
		name         string
		input        workflowengine.ActivityInput
		expectError  bool
		expectLog    string
		checkPort    bool
		expectedPort string
	}{
		{
			name: "Success -valid input",
			input: workflowengine.ActivityInput{
				Payload: map[string]any{
					"image": "alpine:latest",
					"cmd":   []string{"echo", "hello world"},
				},
				Config: map[string]string{},
			},
			expectError: false,
			expectLog:   "hello",
		},
		{
			name: "Success - environment variables set",
			input: workflowengine.ActivityInput{
				Payload: map[string]any{
					"image": "alpine:latest",
					"cmd":   []string{"sh", "-c", "echo $FOO"},
					"env":   []string{"FOO=bar"},
				},
				Config: map[string]string{},
			},
			expectError: false,
			expectLog:   "bar",
		},
		{
			name: "Success - port exposed",
			input: workflowengine.ActivityInput{
				Payload: map[string]any{
					"image": "alpine:latest",
					"cmd":   []string{"sh", "-c", "sleep 5"},
					"ports": []string{"8080:80"},
				},
				Config: map[string]string{
					"HostIP": "127.0.0.1",
				},
			},
			expectError:  false,
			checkPort:    true,
			expectedPort: "80/tcp",
		},
		{
			name: "Success - custom network config (bridge)",
			input: workflowengine.ActivityInput{
				Payload: map[string]any{
					"image": "alpine:latest",
					"cmd":   []string{"sh", "-c", "sleep 2"},
					"networkConfig": &network.NetworkingConfig{
						EndpointsConfig: map[string]*network.EndpointSettings{
							"bridge": {},
						},
					},
				},
				Config: map[string]string{
					"KeepContainer": "true",
				},
			},
			expectError: false,
			expectLog:   "", // We're not checking log here
		},
		{
			name: "Failure - missing image",
			input: workflowengine.ActivityInput{
				Payload: map[string]any{},
			},
			expectError: true,
		},
		{
			name: "Failure - invalid port mapping",
			input: workflowengine.ActivityInput{
				Payload: map[string]any{
					"image": "alpine:latest",
					"ports": []string{"8080"},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result workflowengine.ActivityResult
			future, err := env.ExecuteActivity(activity.Execute, tt.input)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				err := future.Get(&result)
				require.NoError(t, err)

				logContent := strings.Join(result.Log, "\n")
				require.Contains(t, logContent, tt.expectLog)

				// Ensure container ID is available
				containerID, ok := result.Output.(map[string]any)["containerID"].(string)
				require.True(t, ok)
				require.NotEmpty(t, containerID)
				cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
				if tt.checkPort {
					ctx := context.Background()
					inspect, err := cli.ContainerInspect(ctx, containerID)
					require.NoError(t, err)
					_, ok := inspect.HostConfig.PortBindings[nat.Port(tt.expectedPort)]
					require.True(t, ok, "Expected port %s to be exposed", tt.expectedPort)
				}

				if tt.name == "Success - custom network config (bridge)" {
					require.NoError(t, err)
					ctx := context.Background()
					inspect, err := cli.ContainerInspect(ctx, containerID)
					require.NoError(t, err)

					// Check if "bridge" network is attached
					_, ok := inspect.NetworkSettings.Networks["bridge"]
					require.True(t, ok, "Expected container to be connected to 'bridge' network")
				}
				cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
			}
		})
	}
}

func TestBuildPortMappings(t *testing.T) {
	tests := []struct {
		name                 string
		hostIP               string
		ports                []string
		expectedErr          bool
		expectedExposedPorts nat.PortSet
		expectedPortBindings nat.PortMap
	}{
		{
			name:        "Valid port mappings",
			hostIP:      "0.0.0.0",
			ports:       []string{"8080:80", "9090:90"},
			expectedErr: false,
			expectedExposedPorts: nat.PortSet{
				"80/tcp": {},
				"90/tcp": {},
			},
			expectedPortBindings: nat.PortMap{
				"80/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "8080"}},
				"90/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "9090"}},
			},
		},
		{
			name:                 "Invalid port mapping format",
			hostIP:               "0.0.0.0",
			ports:                []string{"8080", "9090:90"},
			expectedErr:          true,
			expectedExposedPorts: nil,
			expectedPortBindings: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exposedPorts, portBindings, err := buildPortMappings(tt.hostIP, tt.ports)

			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedExposedPorts, exposedPorts)
				require.Equal(t, tt.expectedPortBindings, portBindings)
			}
		})
	}
}
