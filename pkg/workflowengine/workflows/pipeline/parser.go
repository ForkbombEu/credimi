// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ParseWorkflow parses YAML string into Workflow struct
func ParseWorkflow(yamlStr string) (*WorkflowDefinition, error) {
	var wf WorkflowDefinition
	if err := yaml.Unmarshal([]byte(yamlStr), &wf); err != nil {
		return nil, fmt.Errorf("failed to parse workflow yaml: %w", err)
	}
	if wf.Entry != "" {
		if entryDef, ok := wf.Entries[wf.Entry]; ok {
			wf.Steps = entryDef.Steps
			if wf.Config == nil {
				wf.Config = entryDef.Config
			}
		} else {
			return nil, fmt.Errorf("entry %q not found in workflow", wf.Entry)
		}
	}

	return &wf, nil
}
