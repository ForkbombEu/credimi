// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"context"
	"fmt"
	"log"

	"github.com/forkbombeu/credimi/pkg/internal/apis/handlers"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/pocketbase/core"
	"gopkg.in/yaml.v3"
)

type MobileRunner struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	CanonifiedName string `json:"canonified_name"`
}

type ScheduleStatus struct {
	DisplayName    string         `json:"display_name,omitempty"`
	NextActionTime string         `json:"next_action_time,omitempty"`
	Paused         bool           `json:"paused"`
	Runners        []MobileRunner `json:"runners"`
}

func RegisterSchedulesHooks(app core.App) {
	app.OnRecordEnrich("schedules").BindFunc(func(e *core.RecordEnrichEvent) error {
		ownerID := e.Record.GetString("owner")

		owner, err := e.App.FindRecordById("organizations", ownerID)
		if err != nil {
			return fmt.Errorf("failed to fetch owner organization: %w", err)
		}

		namespace := owner.GetString("canonified_name")

		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return fmt.Errorf(
				"unable to create Temporal client for namespace %q: %w",
				namespace,
				err,
			)
		}
		ctx := context.Background()
		handle := c.ScheduleClient().GetHandle(ctx, e.Record.GetString("temporal_schedule_id"))
		desc, err := handle.Describe(ctx)
		if err != nil {
			return fmt.Errorf("failed to describe schedule: %w", err)
		}
		var displayName string
		if desc.Memo != nil {
			if field, ok := desc.Memo.Fields["test"]; ok {
				displayName = handlers.DecodeFromTemporalPayload(string(field.Data))
			}
		}

		// Parse runners from pipeline yaml
		runners, err := parseRunnersFromPipeline(e.App, e.Record.GetString("pipeline"))
		if err != nil {
			// Log error but don't fail the enrichment
			log.Printf("failed to parse runners from pipeline: %v\n", err)
			runners = []MobileRunner{}
		}

		status := ScheduleStatus{
			DisplayName:    displayName,
			NextActionTime: desc.Info.NextActionTimes[0].Format("02/01/2006, 15:04:05"),
			Paused:         desc.Schedule.State.Paused,
			Runners:        runners,
		}
		e.Record.WithCustomData(true)
		e.Record.Set("__schedule_status__", status)

		return e.Next()

	})

}

// parseRunnersFromPipeline extracts mobile runners from pipeline YAML
func parseRunnersFromPipeline(app core.App, pipelineID string) ([]MobileRunner, error) {
	// Resolve pipeline record
	pipelineRec, err := canonify.Resolve(app, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve pipeline: %w", err)
	}

	// Parse pipeline YAML
	yamlContent := pipelineRec.GetString("yaml")
	var wfDef pipeline.WorkflowDefinition
	if err := yaml.Unmarshal([]byte(yamlContent), &wfDef); err != nil {
		return nil, fmt.Errorf("failed to parse pipeline yaml: %w", err)
	}

	// Extract runner IDs from mobile-automation steps
	runnerIDsMap := make(map[string]bool)
	for _, step := range wfDef.Steps {
		if step.Use == "mobile-automation" {
			if runnerID, ok := step.With.Config["runner_id"].(string); ok && runnerID != "" {
				runnerIDsMap[runnerID] = true
			}
		}
	}

	// Fetch runner records and build response
	runners := []MobileRunner{}
	for runnerID := range runnerIDsMap {
		runnerRec, err := canonify.Resolve(app, runnerID)
		if err != nil {
			// Skip if runner not found
			continue
		}

		runners = append(runners, MobileRunner{
			ID:             runnerRec.Id,
			Name:           runnerRec.GetString("name"),
			CanonifiedName: runnerRec.GetString("canonified_name"),
		})
	}

	return runners, nil
}
