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

	imageRaw, ok := input.Payload["image"].(string)
	if !ok || imageRaw == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: 'yaml", errCode.Description),
		)
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

	out, err := cli.ImagePull(ctx, imageRaw, image.PullOptions{})
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.DockerPullImageFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			imageRaw,
		)
	}
	defer out.Close()
	io.Copy(io.Discard, out)

	cmd := asSliceOfStrings(input.Payload["cmd"])
	user, _ := input.Payload["user"].(string)
	env := asSliceOfStrings(input.Payload["env"])
	ports := asSliceOfStrings(input.Payload["ports"])
	mounts := asSliceOfStrings(input.Payload["mounts"])
	containerName, ok := input.Payload["containerName"].(string)
	if !ok {
		containerName = ""
	}

	hostIP := input.Config["HostIP"]
	if hostIP == "" {
		hostIP = "0.0.0.0" // Default to "0.0.0.0" if not provided
	}
	exposedPorts, portBindings, err := buildPortMappings(hostIP, ports)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
		)
	}

	var networkConfig *network.NetworkingConfig
	if rawNetworkConfig, ok := input.Payload["networkConfig"].(*network.NetworkingConfig); ok {
		networkConfig = rawNetworkConfig
	}

	config := &container.Config{
		Image:        imageRaw,
		Cmd:          cmd,
		User:         user,
		Env:          env,
		ExposedPorts: exposedPorts,
		AttachStdout: true,
		AttachStderr: true,
	}

	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Binds:        mounts,
	}

	resp, err := cli.ContainerCreate(ctx, config, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.DockerCreateContainerFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			containerName,
			config,
			hostConfig,
			networkConfig,
		)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		errCode := errorcodes.Codes[errorcodes.DockerStartContainerFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			containerName,
			config,
			hostConfig,
			networkConfig,
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

func asSliceOfStrings(val any) []string {
	if v, ok := val.([]string); ok {
		return v
	}
	if arr, ok := val.([]any); ok {
		res := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				res = append(res, s)
			}
		}
		return res
	}
	return nil
}
