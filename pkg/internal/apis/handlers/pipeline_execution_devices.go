// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"strings"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
)

// iosSimulatorDeviceType is excluded from live view (runner contract).
const iosSimulatorDeviceType = "ios_simulator"

// PipelineExecutionDevice is a display-ready device row for pipeline execution UIs.
type PipelineExecutionDevice struct {
	DeviceID string `json:"device_id"`
	Name     string `json:"name"`
	LiveView bool   `json:"live_view"`
}

// DeviceSupportsLiveView reports whether a mobile device type can open live view.
func DeviceSupportsLiveView(deviceType string) bool {
	return strings.TrimSpace(deviceType) != iosSimulatorDeviceType
}

// ResolveAndBuildPipelineExecutionDevices resolves PocketBase device records and
// builds ordered display rows. live_view is true only when status is Running and
// the device type supports live view.
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
		deviceType := ""
		if record != nil {
			deviceType = workflowengine.AsString(record["type"])
		}
		out = append(out, PipelineExecutionDevice{
			DeviceID: deviceID,
			Name:     deviceDisplayName(deviceID, record),
			LiveView: running && DeviceSupportsLiveView(deviceType),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func pipelineExecutionDevicesFromDescribe(
	app core.App,
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

	decoded, err := decodeWorkflowSearchAttributes(info.GetSearchAttributes())
	if err != nil || len(decoded) == 0 {
		return nil
	}
	deviceIDs := deviceIDsFromSearchAttributes(&decoded)
	if len(deviceIDs) == 0 {
		return nil
	}

	status := string(WorkflowStatusCompleted)
	if info.GetStatus() == enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
		status = string(WorkflowStatusRunning)
	}
	return ResolveAndBuildPipelineExecutionDevices(app, deviceIDs, status, nil)
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
