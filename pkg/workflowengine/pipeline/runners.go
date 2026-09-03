// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"sort"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/pocketbase/pocketbase/core"
)

type PipelineDeviceInfo struct {
	DeviceIDs         []string
	NeedsGlobalDevice bool
}

// ValidateDeviceIDYAML enforces the mobile device_id configuration rule.
func ValidateDeviceIDYAML(yamlStr string) error {
	wfDef, err := pipeline.ParseWorkflow(yamlStr)
	if err != nil {
		return err
	}

	return validateMobileDeviceIDConfiguration(wfDef.Steps, wfDef.Runtime.GlobalDeviceID)
}

func ParsePipelineDeviceInfo(yamlStr string) (PipelineDeviceInfo, error) {
	if strings.TrimSpace(yamlStr) == "" {
		return PipelineDeviceInfo{}, nil
	}

	wfDef, err := pipeline.ParseWorkflow(yamlStr)
	if err != nil {
		return PipelineDeviceInfo{}, err
	}

	deviceIDs := make(map[string]struct{})
	missingDeviceID := false

	collectDevice := func(step pipeline.StepSpec) {
		deviceID := ""
		if step.With.Payload != nil {
			if rawDeviceID, ok := step.With.Payload["device_id"]; ok {
				if id, ok := rawDeviceID.(string); ok {
					deviceID = canonify.NormalizePath(id)
				}
			}
		}

		if deviceID != "" {
			deviceIDs[deviceID] = struct{}{}
			return
		}

		if step.Use == mobileAutomationStepUse {
			missingDeviceID = true
		}
	}

	for _, step := range wfDef.Steps {
		collectDevice(step.StepSpec)
		for _, onErr := range step.OnError {
			collectDevice(onErr.StepSpec)
		}
		for _, onSuccess := range step.OnSuccess {
			collectDevice(onSuccess.StepSpec)
		}
	}

	info := PipelineDeviceInfo{
		NeedsGlobalDevice: missingDeviceID,
	}

	if len(deviceIDs) == 0 {
		return info, nil
	}

	info.DeviceIDs = make([]string, 0, len(deviceIDs))
	for deviceID := range deviceIDs {
		info.DeviceIDs = append(info.DeviceIDs, deviceID)
	}
	sort.Strings(info.DeviceIDs)

	return info, nil
}

func DeviceIDsWithGlobal(info PipelineDeviceInfo, globalDeviceID string) []string {
	deviceIDs := make([]string, 0, len(info.DeviceIDs))
	for _, deviceID := range info.DeviceIDs {
		deviceID = canonify.NormalizePath(deviceID)
		if deviceID == "" {
			continue
		}
		deviceIDs = append(deviceIDs, deviceID)
	}
	globalDeviceID = canonify.NormalizePath(globalDeviceID)
	if info.NeedsGlobalDevice && globalDeviceID != "" {
		found := false
		for _, id := range deviceIDs {
			if id == globalDeviceID {
				found = true
				break
			}
		}
		if !found {
			deviceIDs = append(deviceIDs, globalDeviceID)
			sort.Strings(deviceIDs)
		}
	}

	return deviceIDs
}

func GlobalDeviceIDFromConfig(config map[string]any) string {
	if config == nil {
		return ""
	}
	if v, ok := config["global_device_id"]; ok {
		if s, ok := v.(string); ok {
			return canonify.NormalizePath(s)
		}
	}
	return ""
}

func ResolveDeviceRecord(
	app core.App,
	deviceID string,
	deviceCache map[string]map[string]any,
) map[string]any {
	if deviceCache == nil {
		deviceCache = map[string]map[string]any{}
	}

	deviceID = canonify.NormalizePath(deviceID)
	if deviceID == "" {
		return nil
	}

	if cached, ok := deviceCache[deviceID]; ok {
		return cached
	}

	record, err := canonify.Resolve(app, deviceID)
	if err != nil || record.Collection() == nil || record.Collection().Name != "mobile_devices" {
		deviceCache[deviceID] = nil
		return nil
	}

	fields := record.PublicExport()
	fields[core.FieldNameId] = record.Id
	deviceCache[deviceID] = fields
	return fields
}

func ResolveDeviceRecords(
	app core.App,
	deviceIDs []string,
	deviceCache map[string]map[string]any,
) []map[string]any {
	if len(deviceIDs) == 0 {
		return []map[string]any{}
	}

	records := make([]map[string]any, 0, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		record := ResolveDeviceRecord(app, deviceID, deviceCache)
		if record == nil {
			continue
		}
		records = append(records, record)
	}
	return records
}
