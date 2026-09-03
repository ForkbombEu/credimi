// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	internalpipeline "github.com/forkbombeu/credimi/pkg/internal/pipeline"
)

func validateMobileDeviceIDConfiguration(
	steps []internalpipeline.StepDefinition,
	globalDeviceID string,
) error {
	globalSet := canonify.NormalizePath(globalDeviceID) != ""

	return walkMutableStepSpecs(steps, func(step *internalpipeline.StepSpec) error {
		if step.Use != mobileAutomationStepUse {
			return nil
		}

		deviceID, _ := step.With.Payload["device_id"].(string)
		deviceSet := canonify.NormalizePath(deviceID) != ""
		if globalSet && deviceSet {
			return fmt.Errorf(
				"global_device_id is set, but step %q defines device_id; use only global_device_id or set device_id for all mobile-automation steps",
				step.ID,
			)
		}
		if !globalSet && !deviceSet {
			return fmt.Errorf(
				"global_device_id is not set and step %q is missing device_id; set device_id on all mobile-automation steps or set global_device_id",
				step.ID,
			)
		}
		return nil
	})
}
