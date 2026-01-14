// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

type fakeDockerClient struct {
	createdConfig        *container.Config
	createdHostConfig    *container.HostConfig
	createdNetworkConfig *network.NetworkingConfig
	containerID          string
	exitCode             int
	stdout               string
	stderr               string
	waitErr              error
}

func (client *fakeDockerClient) ImagePull(
	_ context.Context,
	_ string,
	_ image.PullOptions,
) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (client *fakeDockerClient) ContainerCreate(
	_ context.Context,
	config *container.Config,
	hostConfig *container.HostConfig,
	networkingConfig *network.NetworkingConfig,
	_ *ocispec.Platform,
	_ string,
) (container.CreateResponse, error) {
	client.createdConfig = config
	client.createdHostConfig = hostConfig
	client.createdNetworkConfig = networkingConfig
	if client.containerID == "" {
		client.containerID = "container-test"
	}
	return container.CreateResponse{ID: client.containerID}, nil
}

func (client *fakeDockerClient) ContainerStart(
	_ context.Context,
	_ string,
	_ container.StartOptions,
) error {
	return nil
}

func (client *fakeDockerClient) ContainerWait(
	_ context.Context,
	_ string,
	_ container.WaitCondition,
) (<-chan container.WaitResponse, <-chan error) {
	statusCh := make(chan container.WaitResponse, 1)
	errCh := make(chan error, 1)
	if client.waitErr != nil {
		errCh <- client.waitErr
	} else {
		statusCh <- container.WaitResponse{StatusCode: int64(client.exitCode)}
	}
	close(statusCh)
	close(errCh)
	return statusCh, errCh
}

func (client *fakeDockerClient) ContainerInspect(
	_ context.Context,
	_ string,
) (container.InspectResponse, error) {
	return container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			State:      &container.State{ExitCode: client.exitCode},
			HostConfig: client.createdHostConfig,
		},
		NetworkSettings: &container.NetworkSettings{
			Networks: map[string]*network.EndpointSettings{
				"bridge": {},
			},
		},
	}, nil
}

func (client *fakeDockerClient) ContainerLogs(
	_ context.Context,
	_ string,
	_ container.LogsOptions,
) (io.ReadCloser, error) {
	var buf bytes.Buffer
	_, _ = stdcopy.NewStdWriter(&buf, stdcopy.Stdout).Write([]byte(client.stdout))
	_, _ = stdcopy.NewStdWriter(&buf, stdcopy.Stderr).Write([]byte(client.stderr))
	return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

func (client *fakeDockerClient) ContainerKill(
	_ context.Context,
	_ string,
	_ string,
) error {
	return nil
}

func (client *fakeDockerClient) ContainerRemove(
	_ context.Context,
	_ string,
	_ container.RemoveOptions,
) error {
	return nil
}

func (client *fakeDockerClient) Close() error {
	return nil
}

func TestDockerRunActivity_Execute(t *testing.T) {
	act := NewDockerActivity()
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()
	env.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{
		Name: act.Name(),
	})

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
				Payload: DockerActivityPayload{
					Image: "alpine:latest",
					Cmd:   []string{"echo", "hello world"},
				},
				Config: map[string]string{},
			},
			expectError: false,
			expectLog:   "hello",
		},
		{
			name: "Success - environment variables set",
			input: workflowengine.ActivityInput{
				Payload: DockerActivityPayload{
					Image: "alpine:latest",
					Cmd:   []string{"sh", "-c", "echo $FOO"},
					Env:   []string{"FOO=bar"},
				},
				Config: map[string]string{},
			},
			expectError: false,
			expectLog:   "bar",
		},
		{
			name: "Success - port exposed",
			input: workflowengine.ActivityInput{
				Payload: DockerActivityPayload{
					Image: "alpine:latest",
					Cmd:   []string{"sh", "-c", "sleep 5"},
					Ports: []string{"8080:80"},
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
				Payload: DockerActivityPayload{
					Image: "alpine:latest",
					Cmd:   []string{"sh", "-c", "sleep 2"},
					NetworkConfig: &network.NetworkingConfig{
						EndpointsConfig: map[string]*network.EndpointSettings{
							"bridge": {},
						},
					},
				},
			},
			expectError: false,
			expectLog:   "", // We're not checking log here
		},
		{
			name: "Failure - missing image",
			input: workflowengine.ActivityInput{
				Payload: DockerActivityPayload{},
			},
			expectError: true,
		},
		{
			name: "Failure - invalid port mapping",
			input: workflowengine.ActivityInput{
				Payload: DockerActivityPayload{
					Image: "alpine:latest",
					Ports: []string{"8080"},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeDockerClient{
				exitCode: 0,
				stdout:   tt.expectLog,
			}
			originalFactory := dockerClientFactory
			dockerClientFactory = func() (dockerClient, error) {
				return fake, nil
			}
			t.Cleanup(func() {
				dockerClientFactory = originalFactory
			})

			var result workflowengine.ActivityResult
			future, err := env.ExecuteActivity(act.Execute, tt.input)

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

				if tt.checkPort {
					require.NotNil(t, fake.createdHostConfig)
					_, ok := fake.createdHostConfig.PortBindings[nat.Port(tt.expectedPort)]
					require.True(t, ok, "Expected port %s to be exposed", tt.expectedPort)
				}

				if tt.name == "Success - custom network config (bridge)" {
					require.NotNil(t, fake.createdNetworkConfig)
					_, ok := fake.createdNetworkConfig.EndpointsConfig["bridge"]
					require.True(t, ok, "Expected container to be connected to 'bridge' network")
				}
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
