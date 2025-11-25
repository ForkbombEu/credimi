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
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
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
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
		)
	}
	defer cli.Close()

	out, err := cli.ImagePull(ctx, payload.Image, image.PullOptions{})
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.DockerPullImageFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			payload.Image,
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
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
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

	resp, err := cli.ContainerCreate(
		ctx,
		config,
		hostConfig,
		payload.NetworkConfig,
		nil,
		payload.ContainerName,
	)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.DockerCreateContainerFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			payload.ContainerName,
			config,
			hostConfig,
			payload.NetworkConfig,
		)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		errCode := errorcodes.Codes[errorcodes.DockerStartContainerFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			resp.ID,
			config,
			hostConfig,
			payload.NetworkConfig,
		)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			errCode := errorcodes.Codes[errorcodes.DockerWaitContainerFailed]
			return result, a.NewActivityError(
				errCode.Code,
				fmt.Sprintf("%s: %v", errCode.Description, err),
				resp.ID,
			)
		}
	case <-statusCh:
	}

	inspect, err := cli.ContainerInspect(ctx, resp.ID)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.DockerInspectContainerFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			resp.ID,
		)
	}

	// Collect logs
	logs, err := cli.ContainerLogs(
		ctx,
		resp.ID,
		container.LogsOptions{ShowStdout: true, ShowStderr: true},
	)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.DockerFetchLogsFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			resp.ID,
		)
	}
	defer logs.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	var combinedBuf bytes.Buffer

	multiStdout := io.MultiWriter(&stdoutBuf, &combinedBuf)
	multiStderr := io.MultiWriter(&stderrBuf, &combinedBuf)

	_, err = stdcopy.StdCopy(multiStdout, multiStderr, logs)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.CopyFromReaderFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
		)
	}

	if inspect.State.ExitCode != 0 {
		errCode := errorcodes.Codes[errorcodes.CommandExecutionFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("Docker command failed with exit code %d", inspect.State.ExitCode),
			resp.ID,
			stdoutBuf.String(),
			stderrBuf.String(),
		)
	}
	result.Log = append(result.Log, combinedBuf.String())
	result.Output = map[string]any{
		"containerID": resp.ID,
		"stdout":      stdoutBuf.String(),
		"stderr":      stderrBuf.String(),
		"exitCode":    inspect.State.ExitCode,
	}
	return result, nil
}

// buildPortMappings takes a slice of port mappings as strings (e.g., "8080:80") and returns
// the Docker port bindings and exposed ports.
func buildPortMappings(hostIP string, ports []string) (nat.PortSet, nat.PortMap, error) {
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}

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

		exposedPort := nat.Port(containerPort + "/tcp")
		exposedPorts[exposedPort] = struct{}{}

		portBindings[exposedPort] = []nat.PortBinding{
			{
				HostIP:   hostIP,
				HostPort: hostPort,
			},
		}
	}

	return exposedPorts, portBindings, nil
}
