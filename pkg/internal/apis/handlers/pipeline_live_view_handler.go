// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

const (
	runnerLiveViewTimeout    = 15 * time.Second
	runnerLiveViewPathPrefix = "/live/"
	iosSimulatorDeviceType   = "ios_simulator"
)

type PipelineLiveViewInput struct {
	WorkflowID string `json:"workflow_id"         validate:"required"`
	RunID      string `json:"run_id"              validate:"required"`
	DeviceID   string `json:"device_id,omitempty"`
}

type PipelineLiveViewStream struct {
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	URL        string `json:"url"`
}

type PipelineLiveViewResponse struct {
	Streams []PipelineLiveViewStream `json:"streams"`
}

type runnerLiveViewRequest struct {
	DeviceIdentifier string `json:"device_identifier"`
	Serial           string `json:"serial"`
	Namespace        string `json:"namespace"`
	WorkflowID       string `json:"workflow_id"`
	RunID            string `json:"run_id"`
}

type runnerLiveViewResponse struct {
	Path string `json:"path"`
}

type runnerLiveViewError struct {
	Message string `json:"message"`
}

var (
	pipelineLiveViewTemporalClient = temporalclient.GetTemporalClientWithNamespace
	openRunnerLiveView             = openRunnerLiveViewHTTP
)

func HandlePipelineLiveView() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[PipelineLiveViewInput](e)
		if err != nil {
			return err
		}
		if e.Auth == nil {
			return apierror.New(
				http.StatusUnauthorized,
				"auth",
				"authentication required",
				"user not authenticated",
			)
		}

		namespace, err := pbutils.GetUserOrganizationCanonifiedName(e.App, e.Auth.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			)
		}

		temporalClient, err := pipelineLiveViewTemporalClient(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			)
		}

		ctx := e.Request.Context()
		workflowID := strings.TrimSpace(input.WorkflowID)
		runID := strings.TrimSpace(input.RunID)
		devices, apiErr := runningPipelineDevices(ctx, temporalClient, workflowID, runID)
		if apiErr != nil {
			return apiErr
		}

		explicit := strings.TrimSpace(input.DeviceID) != ""
		deviceIDs, apiErr := selectLiveViewDevices(devices, input.DeviceID)
		if apiErr != nil {
			return apiErr
		}

		streams := make([]PipelineLiveViewStream, 0, len(deviceIDs))
		var firstErr *apierror.APIError
		for _, deviceID := range deviceIDs {
			stream, apiErr := openPipelineDeviceLiveView(
				ctx,
				deviceID,
				devices[deviceID],
				runnerLiveViewRequest{
					Namespace:  namespace,
					WorkflowID: workflowID,
					RunID:      runID,
				},
			)
			if apiErr != nil {
				if explicit {
					return apiErr
				}
				if firstErr == nil {
					firstErr = apiErr
				}
				continue
			}
			streams = append(streams, stream)
		}
		if len(streams) == 0 && firstErr != nil {
			return firstErr
		}

		return e.JSON(http.StatusOK, PipelineLiveViewResponse{Streams: streams})
	}
}

func runningPipelineDevices(
	ctx context.Context,
	temporalClient client.Client,
	workflowID string,
	runID string,
) (map[string]map[string]any, *apierror.APIError) {
	description, err := temporalClient.DescribeWorkflowExecution(ctx, workflowID, runID)
	if err != nil {
		notFound := &serviceerror.NotFound{}
		if errors.As(err, &notFound) {
			return nil, apierror.New(
				http.StatusNotFound,
				"workflow",
				"pipeline workflow not found",
				err.Error(),
			)
		}
		return nil, apierror.New(
			http.StatusInternalServerError,
			"workflow",
			"unable to describe pipeline workflow",
			err.Error(),
		)
	}
	info := description.GetWorkflowExecutionInfo()
	if info == nil || info.GetStatus() != enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
		return nil, apierror.New(
			http.StatusConflict,
			"workflow",
			"pipeline workflow is not running",
			"live view is available only while the pipeline execution is running",
		)
	}
	if info.GetType() == nil || info.GetType().GetName() != "Dynamic Pipeline Workflow" {
		return nil, apierror.New(
			http.StatusUnprocessableEntity,
			"workflow",
			"workflow is not a dynamic pipeline",
			"live view is available only for dynamic pipeline workflows",
		)
	}

	encoded, err := temporalClient.QueryWorkflow(
		ctx,
		workflowID,
		runID,
		pipeline.PipelineMobileDevicesQuery,
	)
	if err != nil {
		return nil, apierror.New(
			http.StatusConflict,
			"device_id",
			"pipeline mobile device is not initialized",
			err.Error(),
		)
	}
	var raw map[string]any
	if err := encoded.Get(&raw); err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"device_id",
			"failed to read initialized pipeline devices",
			err.Error(),
		)
	}

	devices := make(map[string]map[string]any, len(raw))
	for id, value := range raw {
		if device, ok := value.(map[string]any); ok {
			devices[id] = device
		}
	}

	return devices, nil
}

func selectLiveViewDevices(
	devices map[string]map[string]any,
	requestedID string,
) ([]string, *apierror.APIError) {
	if strings.TrimSpace(requestedID) != "" {
		deviceID := canonify.NormalizePath(requestedID)
		device, ok := devices[deviceID]
		if !ok {
			return nil, apierror.New(
				http.StatusConflict,
				"device_id",
				"pipeline mobile device is not initialized",
				"the requested device is not initialized in this pipeline execution",
			)
		}
		if workflowengine.AsString(device["type"]) == iosSimulatorDeviceType {
			return nil, apierror.New(
				http.StatusUnprocessableEntity,
				"device_id",
				"live view is not supported for ios_simulator devices",
				"live view is available only for Android devices",
			)
		}
		return []string{deviceID}, nil
	}

	deviceIDs := make([]string, 0, len(devices))
	for id, device := range devices {
		if workflowengine.AsString(device["type"]) == iosSimulatorDeviceType {
			continue
		}
		deviceIDs = append(deviceIDs, id)
	}
	if len(deviceIDs) == 0 {
		return nil, apierror.New(
			http.StatusConflict,
			"device_id",
			"no live-view capable device is initialized",
			"live view is available only for Android devices",
		)
	}
	sort.Strings(deviceIDs)

	return deviceIDs, nil
}

func openPipelineDeviceLiveView(
	ctx context.Context,
	deviceID string,
	device map[string]any,
	body runnerLiveViewRequest,
) (PipelineLiveViewStream, *apierror.APIError) {
	runnerURL := workflowengine.AsString(device["runner_url"])
	if !mobilerunner.URLUsable(runnerURL) {
		return PipelineLiveViewStream{}, apierror.New(
			http.StatusConflict,
			"runner_url",
			"device runner URL is not usable",
			"the runner holding this device has no callable URL",
		)
	}

	body.DeviceIdentifier = deviceID
	body.Serial = workflowengine.AsString(device["serial"])
	streamURL, apiErr := openRunnerLiveView(ctx, runnerURL, body)
	if apiErr != nil {
		return PipelineLiveViewStream{}, apiErr
	}

	name := deviceID
	if index := strings.LastIndex(deviceID, "/"); index >= 0 {
		name = deviceID[index+1:]
	}

	return PipelineLiveViewStream{DeviceID: deviceID, DeviceName: name, URL: streamURL}, nil
}

func openRunnerLiveViewHTTP(
	ctx context.Context,
	runnerURL string,
	body runnerLiveViewRequest,
) (string, *apierror.APIError) {
	apiKey := strings.TrimSpace(os.Getenv(InternalAdminAPIKeyEnvVar))
	if apiKey == "" {
		return "", apierror.New(
			http.StatusInternalServerError,
			"live_view",
			"internal admin key is not configured",
			InternalAdminAPIKeyEnvVar+" is empty",
		)
	}

	endpoint, err := url.JoinPath(runnerURL, "credimi", "live-view")
	if err != nil {
		return "", apierror.New(
			http.StatusConflict,
			"runner_url",
			"device runner URL is not usable",
			err.Error(),
		)
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"live_view",
			"failed to encode live view request",
			err.Error(),
		)
	}

	requestCtx, cancel := context.WithTimeout(ctx, runnerLiveViewTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(payload),
	)
	if err != nil {
		return "", apierror.New(
			http.StatusConflict,
			"runner_url",
			"device runner URL is not usable",
			err.Error(),
		)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(APIKeyHeaderName, apiKey)

	resp, err := mobilerunner.HTTPClient(runnerURL).Do(req)
	if err != nil {
		return "", apierror.New(
			http.StatusServiceUnavailable,
			"live_view",
			"device runner is offline",
			err.Error(),
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var runnerErr runnerLiveViewError
		_ = json.NewDecoder(resp.Body).Decode(&runnerErr)
		message := strings.TrimSpace(runnerErr.Message)
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		if resp.StatusCode == http.StatusServiceUnavailable {
			return "", apierror.New(
				http.StatusServiceUnavailable,
				"live_view",
				"live view is unavailable on the device runner",
				message,
			)
		}
		return "", apierror.New(
			http.StatusBadGateway,
			"live_view",
			"runner refused live view",
			message,
		)
	}

	var result runnerLiveViewResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil ||
		!strings.HasPrefix(result.Path, runnerLiveViewPathPrefix) {
		return "", apierror.New(
			http.StatusBadGateway,
			"live_view",
			"runner refused live view",
			"runner returned an invalid live view path",
		)
	}

	return strings.TrimRight(runnerURL, "/") + result.Path, nil
}
