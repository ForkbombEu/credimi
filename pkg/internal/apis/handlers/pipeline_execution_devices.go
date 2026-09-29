// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	temporalcrypto "github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/pocketbase/core"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

// iosSimulatorDeviceType is excluded from live view (runner contract).
const iosSimulatorDeviceType = "ios_simulator"

// PipelineExecutionDevice is a display-ready device row for pipeline execution UIs.
type PipelineExecutionDevice struct {
	DeviceID string `json:"device_id"`
	Name     string `json:"name"`
	LiveView bool   `json:"live_view"`
}

// ResolveAndBuildPipelineExecutionDevices resolves PocketBase device records and
// builds ordered display rows. live_view is true only when status is Running and
// the device record has live_stream, which the runner reports when the device
// has live stream enabled and the runner can serve it.
func ResolveAndBuildPipelineExecutionDevices(
	app core.App,
	deviceIDs []string,
	status string,
	deviceCache map[string]map[string]any,
) []PipelineExecutionDevice {
	if len(deviceIDs) == 0 {
		return nil
	}
	if deviceCache == nil {
		deviceCache = map[string]map[string]any{}
	}

	running := status == string(WorkflowStatusRunning)
	out := make([]PipelineExecutionDevice, 0, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		deviceID = strings.TrimSpace(deviceID)
		if deviceID == "" {
			continue
		}
		record := pipeline.ResolveDeviceRecord(app, deviceID, deviceCache)
		out = append(out, pipelineExecutionDevice(deviceID, record, running))
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func pipelineExecutionDevice(
	deviceID string,
	record map[string]any,
	running bool,
) PipelineExecutionDevice {
	liveStream, _ := record["live_stream"].(bool)
	return PipelineExecutionDevice{
		DeviceID: deviceID,
		Name:     deviceDisplayName(deviceID, record),
		LiveView: running && liveStream,
	}
}

// pipelineExecutionDevicesFromDescribe builds display devices for GET workflow.
// Prefer DeviceIdentifiers search attributes. When those are missing (common when
// the attribute is unset or decode of unrelated attrs fails), fall back to the
// workflow start input — the same YAML + global_device_id resolution list-executions uses.
func pipelineExecutionDevicesFromDescribe(
	ctx context.Context,
	app core.App,
	namespace string,
	exec *workflowservice.DescribeWorkflowExecutionResponse,
) []PipelineExecutionDevice {
	if exec == nil {
		return nil
	}
	info := exec.GetWorkflowExecutionInfo()
	if info == nil || info.GetType() == nil {
		return nil
	}
	if info.GetType().GetName() != pipeline.NewPipelineWorkflow().Name() {
		return nil
	}

	deviceIDs := deviceIDsFromDescribeSearchAttributes(info.GetSearchAttributes())
	if len(deviceIDs) == 0 {
		if we := info.GetExecution(); we != nil {
			deviceIDs = deviceIDsFromWorkflowStartInput(
				ctx,
				namespace,
				we.GetWorkflowId(),
				we.GetRunId(),
			)
		}
	}
	if len(deviceIDs) == 0 {
		return nil
	}

	status := string(WorkflowStatusCompleted)
	if info.GetStatus() == enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
		status = string(WorkflowStatusRunning)
	}
	return ResolveAndBuildPipelineExecutionDevices(app, deviceIDs, status, nil)
}

func deviceIDsFromDescribeSearchAttributes(
	searchAttributes *commonpb.SearchAttributes,
) []string {
	decoded, err := decodeWorkflowSearchAttributes(searchAttributes)
	if err == nil && len(decoded) > 0 {
		if ids := deviceIDsFromSearchAttributes(&decoded); len(ids) > 0 {
			return ids
		}
	}
	// Direct field decode: don't lose DeviceIdentifiers when another attr fails.
	return deviceIDsFromSearchAttributeField(searchAttributes)
}

func deviceIDsFromSearchAttributeField(
	searchAttributes *commonpb.SearchAttributes,
) []string {
	if searchAttributes == nil {
		return nil
	}
	payload := searchAttributes.GetIndexedFields()[workflowengine.DeviceIdentifiersSearchAttribute]
	if payload == nil {
		return nil
	}
	var value any
	if err := temporalcrypto.DataConverter().FromPayload(payload, &value); err != nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		ids := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok && s != "" {
				ids = append(ids, s)
			}
		}
		return ids
	default:
		return nil
	}
}

func deviceDisplayName(deviceID string, record map[string]any) string {
	if record != nil {
		if name := strings.TrimSpace(workflowengine.AsString(record["name"])); name != "" {
			return name
		}
	}
	if index := strings.LastIndex(deviceID, "/"); index >= 0 && index+1 < len(deviceID) {
		return deviceID[index+1:]
	}
	return deviceID
}

func deviceIDsFromWorkflowStartInput(
	ctx context.Context,
	namespace, workflowID, runID string,
) []string {
	if strings.TrimSpace(workflowID) == "" {
		return nil
	}
	c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
	if err != nil || c == nil {
		return nil
	}
	return deviceIDsFromTemporalHistory(ctx, c, workflowID, runID)
}

func deviceIDsFromTemporalHistory(
	ctx context.Context,
	c client.Client,
	workflowID, runID string,
) []string {
	in, ok, err := readPipelineWorkflowInputFromHistory(ctx, c, workflowID, runID)
	if err != nil || !ok {
		return nil
	}
	info := pipeline.DeviceInfoFromDefinition(in.WorkflowDefinition)
	global := pipeline.GlobalDeviceIDFromConfig(in.WorkflowInput.Config)
	if global == "" && in.WorkflowDefinition != nil {
		global = strings.TrimSpace(in.WorkflowDefinition.Runtime.GlobalDeviceID)
	}
	return pipeline.DeviceIDsWithGlobal(info, global)
}
