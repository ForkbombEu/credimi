// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"fmt"
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

// ValidateDeviceIDYAML enforces device_id configuration rules for mobile-automation steps:
// - If global_device_id is set, no step may define device_id.
// - If global_device_id is not set, every mobile-automation step must define device_id.
func ValidateDeviceIDYAML(yamlStr string) error {
	wfDef, err := pipeline.ParseWorkflow(yamlStr)
	if err != nil {
		return err
	}

	globalDeviceID := strings.TrimSpace(wfDef.Runtime.GlobalDeviceID)
	globalSet := globalDeviceID != ""

	foundMobileStep := false

	firstConflictStepID := ""
	firstMissingStepID := ""

	anyStepDeviceSet := false
	anyStepDeviceMissing := false

	for _, step := range wfDef.Steps {
		if step.Use != mobileAutomationStepUse {
			continue
		}

		foundMobileStep = true

		deviceID, _ := step.With.Payload["device_id"].(string)
		deviceSet := strings.TrimSpace(deviceID) != ""

		if deviceSet {
			anyStepDeviceSet = true
			if globalSet && firstConflictStepID == "" {
				firstConflictStepID = step.ID
			}
		} else {
			anyStepDeviceMissing = true
			if !globalSet && firstMissingStepID == "" {
				firstMissingStepID = step.ID
			}
		}
	}

	// No mobile-automation steps → nothing to validate.
	if !foundMobileStep {
		return nil
	}

	// If global is set, no mobile step may set runner_id.
	if globalSet {
		if anyStepDeviceSet {
			return fmt.Errorf(
				"global_device_id is set, but step %q defines device_id; use only global_device_id or set device_id for all mobile-automation steps",
				firstConflictStepID,
			)
		}
		return nil
	}

	// If global is not set, all mobile steps must set runner_id.
	if anyStepDeviceMissing {
		return fmt.Errorf(
			"global_device_id is not set and step %q is missing device_id; set device_id on all mobile-automation steps or set global_device_id",
			firstMissingStepID,
		)
	}

	return nil
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
	if err != nil {
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

// Deprecated compatibility aliases are kept only while internal callers are
// migrated in this implementation slice. They are not serialized contracts.
type PipelineRunnerInfo = PipelineDeviceInfo

func ValidateRunnerIDYAML(yamlStr string) error { return ValidateDeviceIDYAML(yamlStr) }

func ParsePipelineRunnerInfo(yamlStr string) (PipelineDeviceInfo, error) {
	return ParsePipelineDeviceInfo(yamlStr)
}

func RunnerIDsWithGlobal(info PipelineDeviceInfo, globalDeviceID string) []string {
	return DeviceIDsWithGlobal(info, globalDeviceID)
}

func ResolveRunnerRecord(app core.App, deviceID string, cache map[string]map[string]any) map[string]any {
	return ResolveDeviceRecord(app, deviceID, cache)
}

func ResolveRunnerRecords(app core.App, deviceIDs []string, cache map[string]map[string]any) []map[string]any {
	return ResolveDeviceRecords(app, deviceIDs, cache)
}
