// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package activities is a package that provides activities for the workflow engine.
// This file contains the DockerActivity struct and its methods.
package activities

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"go.temporal.io/sdk/activity"
)

// DockerActivity is an activity that runs a Docker image with the specified command and environment variables.
type DockerActivity struct {
	workflowengine.BaseActivity
}

// DockerActivityInput is the input payload for the Docker activity.
type DockerActivityPayload struct {
	Image string `json:"image" yaml:"image" validate:"required"`

	Cmd           []string                  `json:"cmd,omitempty"           yaml:"cmd,omitempty"`
	User          string                    `json:"user,omitempty"          yaml:"user,omitempty"`
	Env           []string                  `json:"env,omitempty"           yaml:"env,omitempty"`
	Ports         []string                  `json:"ports,omitempty"         yaml:"ports,omitempty"`
	Mounts        []string                  `json:"mounts,omitempty"        yaml:"mounts,omitempty"`
	ContainerName string                    `json:"containerName,omitempty" yaml:"containerName,omitempty"`
	NetworkConfig *network.NetworkingConfig `json:"networkConfig,omitempty" yaml:"networkConfig,omitempty"`
}

func NewDockerActivity() *DockerActivity {
	return &DockerActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Run a Docker Image",
		},
	}
}

// Name returns the name of the Docker activity.
func (a *DockerActivity) Name() string {
	return a.BaseActivity.Name
}

// Execute pulls a Docker image, creates a container, and starts it with the provided command and environment variables.
// It also sets up port bindings and collects logs from the container.
// The input payload should contain the following keys:
// - "image": The Docker image to pull (format: "name:version").
// - "cmd": The command to run inside the container (as a slice of strings).
// - "user": The user to run the command as (optional).
// - "env": Environment variables to set inside the container (as a slice of strings).
// - "ports": Port mappings (as a slice of strings, format: "hostPort:containerPort").
// - "containerName": The name of the container (optional).
func (a *DockerActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult

	payload, err := workflowengine.DecodePayload[DockerActivityPayload](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.DockerClientCreationFailed]
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: err.Error(),
			},
		)
	}
	defer cli.Close()

	out, err := cli.ImagePull(ctx, payload.Image, client.ImagePullOptions{})
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.DockerPullImageFailed]
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: err.Error(),
				Details: map[string]any{
					"image": payload.Image,
				},
			},
		)
	}
	defer out.Close()
	io.Copy(io.Discard, out)

	hostIP := input.Config["HostIP"]
	if hostIP == "" {
		hostIP = "0.0.0.0" // Default to "0.0.0.0" if not provided
	}
	exposedPorts, portBindings, err := buildPortMappings(hostIP, payload.Ports)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: err.Error(),
			},
		)
	}

	config := &container.Config{
		Image:        payload.Image,
		Cmd:          payload.Cmd,
		User:         payload.User,
		Env:          payload.Env,
		ExposedPorts: exposedPorts,
		AttachStdout: true,
		AttachStderr: true,
	}

	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Binds:        payload.Mounts,
	}

	resp, err := cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:           config,
		HostConfig:       hostConfig,
		NetworkingConfig: payload.NetworkConfig,
		Name:             payload.ContainerName,
	})
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.DockerCreateContainerFailed]
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: err.Error(),
				Details: map[string]any{
					"container_name": payload.ContainerName,
					"config":         config,
					"host_config":    hostConfig,
					"network_config": payload.NetworkConfig,
				},
			},
		)
	}

	if _, err := cli.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{}); err != nil {
		errCode := errorcodes.Codes[errorcodes.DockerStartContainerFailed]
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: err.Error(),
				Details: map[string]any{
					"container_id":   resp.ID,
					"config":         config,
					"host_config":    hostConfig,
					"network_config": payload.NetworkConfig,
				},
			},
		)
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	waitResult := cli.ContainerWait(
		ctx,
		resp.ID,
		client.ContainerWaitOptions{Condition: container.WaitConditionNotRunning},
	)
	var stdoutBuf, stderrBuf, combinedBuf bytes.Buffer

	for {
		select {
		case <-ctx.Done():
			_, _ = cli.ContainerKill(
				context.Background(),
				resp.ID,
				client.ContainerKillOptions{Signal: "SIGTERM"},
			)
			_, _ = cli.ContainerRemove(
				context.Background(),
				resp.ID,
				client.ContainerRemoveOptions{Force: true},
			)
			return result, ctx.Err()

		case err := <-waitResult.Error:
			if err != nil {
				errCode := errorcodes.Codes[errorcodes.DockerWaitContainerFailed]
				return result, a.NewActivityError(
					workflowengine.ActivityError{
						Code:    errCode.Code,
						Summary: errCode.Description,
						Message: err.Error(),
						Details: map[string]any{
							"container_id": resp.ID,
						},
					},
				)
			}

		case <-waitResult.Result:
			inspect, err := cli.ContainerInspect(ctx, resp.ID, client.ContainerInspectOptions{})
			if err != nil {
				errCode := errorcodes.Codes[errorcodes.DockerInspectContainerFailed]
				return result, a.NewActivityError(
					workflowengine.ActivityError{
						Code:    errCode.Code,
						Summary: errCode.Description,
						Message: err.Error(),
						Details: map[string]any{
							"container_id": resp.ID,
						},
					},
				)
			}

			logs, err := cli.ContainerLogs(
				ctx,
				resp.ID,
				client.ContainerLogsOptions{ShowStdout: true, ShowStderr: true},
			)
			if err != nil {
				errCode := errorcodes.Codes[errorcodes.DockerFetchLogsFailed]
				return result, a.NewActivityError(
					workflowengine.ActivityError{
						Code:    errCode.Code,
						Summary: errCode.Description,
						Message: err.Error(),
						Details: map[string]any{
							"container_id": resp.ID,
						},
					},
				)
			}
			defer logs.Close()

			multiStdout := io.MultiWriter(&stdoutBuf, &combinedBuf)
			multiStderr := io.MultiWriter(&stderrBuf, &combinedBuf)
			_, err = stdcopy.StdCopy(multiStdout, multiStderr, logs)
			if err != nil {
				errCode := errorcodes.Codes[errorcodes.CopyFromReaderFailed]
				return result, a.NewActivityError(
					workflowengine.ActivityError{
						Code:    errCode.Code,
						Summary: errCode.Description,
						Message: err.Error(),
					},
				)
			}

			if inspect.Container.State.ExitCode != 0 {
				errCode := errorcodes.Codes[errorcodes.CommandExecutionFailed]
				return result, a.NewActivityError(
					workflowengine.ActivityError{
						Code:    errCode.Code,
						Summary: "Docker command failed",
						Message: fmt.Sprintf(
							"container exited with code %d",
							inspect.Container.State.ExitCode,
						),
						Category: "external_command",
						Details: map[string]any{
							"container_id": resp.ID,
							"exit_code":    inspect.Container.State.ExitCode,
							"stdout":       stdoutBuf.String(),
							"stderr":       stderrBuf.String(),
						},
					},
				)
			}

			result.Log = append(result.Log, combinedBuf.String())
			result.Output = map[string]any{
				"containerID": resp.ID,
				"stdout":      stdoutBuf.String(),
				"stderr":      stderrBuf.String(),
				"exitCode":    inspect.Container.State.ExitCode,
			}
			return result, nil

		case <-ticker.C:
			activity.RecordHeartbeat(ctx, "")
		}
	}
}

// buildPortMappings takes a slice of port mappings as strings (e.g., "8080:80") and returns
// the Docker port bindings and exposed ports.
func buildPortMappings(hostIP string, ports []string) (network.PortSet, network.PortMap, error) {
	exposedPorts := network.PortSet{}
	portBindings := network.PortMap{}

	if len(ports) == 0 {
		return exposedPorts, portBindings, nil
	}

	hostAddr, err := netip.ParseAddr(hostIP)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid host IP %q: %w", hostIP, err)
	}

	for _, port := range ports {
		// Example: "8080:80"
		parts := strings.Split(port, ":")
		if len(parts) != 2 {
			return nil, nil, errors.New(
				"invalid port mapping format, expected 'hostPort:containerPort'",
			)
		}

		hostPort := parts[0]
		containerPort := parts[1]

		exposedPort, err := network.ParsePort(containerPort + "/tcp")
		if err != nil {
			return nil, nil, err
		}
		exposedPorts[exposedPort] = struct{}{}

		portBindings[exposedPort] = []network.PortBinding{
			{
				HostIP:   hostAddr,
				HostPort: hostPort,
			},
		}
	}

	return exposedPorts, portBindings, nil
}
