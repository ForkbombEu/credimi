// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package runners

import (
	"sort"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/pocketbase/core"
)

type PipelineRunnerInfo struct {
	RunnerIDs         []string
	NeedsGlobalRunner bool
}

func ParsePipelineRunnerInfo(yamlStr string) (PipelineRunnerInfo, error) {
	if strings.TrimSpace(yamlStr) == "" {
		return PipelineRunnerInfo{}, nil
	}

	wfDef, err := pipeline.ParseWorkflow(yamlStr)
	if err != nil {
		return PipelineRunnerInfo{}, err
	}

	runnerIDs := make(map[string]struct{})
	missingRunnerID := false

	collectRunner := func(step pipeline.StepSpec) {
		runnerID := ""
		if step.With.Payload != nil {
			if rawRunnerID, ok := step.With.Payload["runner_id"]; ok {
				if id, ok := rawRunnerID.(string); ok {
					runnerID = strings.TrimSpace(id)
				}
			}
		}

		if runnerID != "" {
			runnerIDs[runnerID] = struct{}{}
			return
		}

		if step.Use == "mobile-automation" {
			missingRunnerID = true
		}
	}

	for _, step := range wfDef.Steps {
		collectRunner(step.StepSpec)
		for _, onErr := range step.OnError {
			collectRunner(onErr.StepSpec)
		}
		for _, onSuccess := range step.OnSuccess {
			collectRunner(onSuccess.StepSpec)
		}
	}

	info := PipelineRunnerInfo{
		NeedsGlobalRunner: missingRunnerID,
	}

	if len(runnerIDs) == 0 {
		return info, nil
	}

	info.RunnerIDs = make([]string, 0, len(runnerIDs))
	for runnerID := range runnerIDs {
		info.RunnerIDs = append(info.RunnerIDs, runnerID)
	}
	sort.Strings(info.RunnerIDs)

	return info, nil
}

func RunnerIDsWithGlobal(info PipelineRunnerInfo, globalRunnerID string) []string {
	runnerIDs := append([]string{}, info.RunnerIDs...)
	globalRunnerID = strings.TrimSpace(globalRunnerID)
	if info.NeedsGlobalRunner && globalRunnerID != "" {
		found := false
		for _, id := range runnerIDs {
			if id == globalRunnerID {
				found = true
				break
			}
		}
		if !found {
			runnerIDs = append(runnerIDs, globalRunnerID)
			sort.Strings(runnerIDs)
		}
	}

	return runnerIDs
}

func GlobalRunnerIDFromConfig(config map[string]any) string {
	if config == nil {
		return ""
	}
	if v, ok := config["global_runner_id"]; ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func ResolveRunnerRecord(
	app core.App,
	runnerID string,
	runnerCache map[string]map[string]any,
) map[string]any {
	if runnerCache == nil {
		runnerCache = map[string]map[string]any{}
	}

	runnerID = strings.TrimSpace(runnerID)
	if runnerID == "" {
		return nil
	}

	if cached, ok := runnerCache[runnerID]; ok {
		return cached
	}

	record, err := canonify.Resolve(app, runnerID)
	if err != nil {
		runnerCache[runnerID] = nil
		return nil
	}

	fields := record.PublicExport()
	fields[core.FieldNameId] = record.Id
	runnerCache[runnerID] = fields
	return fields
}

func ResolveRunnerRecords(
	app core.App,
	runnerIDs []string,
	runnerCache map[string]map[string]any,
) []map[string]any {
	if len(runnerIDs) == 0 {
		return []map[string]any{}
	}

	records := make([]map[string]any, 0, len(runnerIDs))
	for _, runnerID := range runnerIDs {
		record := ResolveRunnerRecord(app, runnerID, runnerCache)
		if record == nil {
			continue
		}
		records = append(records, record)
	}
	return records
}
