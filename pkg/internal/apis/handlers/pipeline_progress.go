// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/utils"
	workflowengine "github.com/forkbombeu/credimi/pkg/workflowengine"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

// progressHistoryLimit caps how many completed runs are inspected per estimate.
const progressHistoryLimit = 25

// progressMaxSamples caps how many durations feed the estimate.
const progressMaxSamples = 10

// PipelineProgress describes the estimated advancement of a running pipeline
// execution, derived from the durations of previous completed runs of the same
// pipeline (runner-matched when possible).
type PipelineProgress struct {
	// ExpectedDurationSeconds is the estimated total duration of the run.
	ExpectedDurationSeconds float64 `json:"expected_duration_seconds"`
	// ElapsedSeconds is the time since the workflow started.
	ElapsedSeconds float64 `json:"elapsed_seconds"`
	// Percent is the estimated completion percentage, capped at 100.
	Percent float64 `json:"percent"`
	// EtaSeconds is the estimated remaining time.
	EtaSeconds float64 `json:"eta_seconds"`
	// SampleSize is the number of previous runs backing the estimate.
	SampleSize int `json:"sample_size"`
}

// computePipelineProgress estimates the progress of a running pipeline
// execution from the durations of previous completed runs of the same
// pipeline, preferring runs executed by the same runners.
func computePipelineProgress(
	ctx context.Context,
	b *pipelineExecutionSummaryBuilder,
	pipelineIdentifier string,
	runnerIDs []string,
	startTime string,
) *PipelineProgress {
	if b == nil || b.client == nil {
		return nil
	}
	normalized := workflowengine.NormalizePipelineIdentifier(pipelineIdentifier)
	if normalized == "" {
		return nil
	}

	start, err := utils.ParseTimeString(startTime)
	if err != nil {
		return nil
	}
	elapsed := time.Since(start)
	if elapsed < 0 {
		return nil
	}

	expected, samples := expectedPipelineDuration(
		ctx,
		b.client,
		b.namespace,
		normalized,
		runnerIDs,
	)
	if expected <= 0 {
		return nil
	}

	percent := elapsed.Seconds() / expected * 100
	if percent > 100 {
		percent = 100
	}
	eta := expected - elapsed.Seconds()
	if eta < 0 {
		eta = 0
	}
	return &PipelineProgress{
		ExpectedDurationSeconds: expected,
		ElapsedSeconds:          elapsed.Seconds(),
		Percent:                 percent,
		EtaSeconds:              eta,
		SampleSize:              samples,
	}
}

// expectedPipelineDuration returns the median duration of recent completed
// runs of the given pipeline. Runs are filtered by runner identifiers when the
// current run declares them and matching history exists.
func expectedPipelineDuration(
	ctx context.Context,
	temporalClient client.Client,
	namespace string,
	pipelineIdentifier string,
	runnerIDs []string,
) (float64, int) {
	query := buildPipelineWorkflowsQuery(
		[]enums.WorkflowExecutionStatus{enums.WORKFLOW_EXECUTION_STATUS_COMPLETED},
		pipelineIdentifier,
	)

	resp, err := temporalClient.ListWorkflow(
		ctx,
		&workflowservice.ListWorkflowExecutionsRequest{
			Namespace: namespace,
			Query:     query,
			PageSize:  progressHistoryLimit,
		},
	)
	if err != nil {
		return 0, 0
	}

	durations := make([]float64, 0, progressMaxSamples)

	for _, execInfo := range resp.GetExecutions() {
		if execInfo == nil || execInfo.GetStartTime() == nil {
			continue
		}
		if len(runnerIDs) > 0 {
			attrs, err := decodeWorkflowSearchAttributes(execInfo.GetSearchAttributes())
			if err != nil || attrs == nil {
				continue
			}
			if historyRunners := runnerIDsFromSearchAttributes(&attrs); len(historyRunners) > 0 &&
				!runnerSetsIntersect(runnerIDs, historyRunners) {
				continue
			}
		}

		closeTime := execInfo.GetCloseTime()
		if closeTime == nil {
			continue
		}
		start := execInfo.GetStartTime()
		if execInfo.GetExecutionTime() != nil {
			start = execInfo.GetExecutionTime()
		}
		duration := closeTime.AsTime().Sub(start.AsTime())
		if duration <= 0 {
			continue
		}
		durations = append(durations, duration.Seconds())
		if len(durations) >= progressMaxSamples {
			break
		}
	}
	if len(durations) == 0 {
		return 0, 0
	}

	sort.Float64s(durations)
	median := durations[len(durations)/2]
	if len(durations)%2 == 0 {
		median = (durations[len(durations)/2-1] + median) / 2
	}
	return median, len(durations)
}

// runnerIDsFromSearchAttributes extracts the RunnerIdentifiers keyword list
// from decoded Temporal search attributes.
func runnerIDsFromSearchAttributes(
	attributes *DecodedWorkflowSearchAttributes,
) []string {
	if attributes == nil {
		return nil
	}
	value, ok := (*attributes)[workflowengine.RunnerIdentifiersSearchAttribute]
	if !ok {
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
	}
	return nil
}

func runnerSetsIntersect(a, b []string) bool {
	set := make(map[string]struct{}, len(a))
	for _, id := range a {
		set[strings.TrimSpace(id)] = struct{}{}
	}
	for _, id := range b {
		if _, ok := set[strings.TrimSpace(id)]; ok {
			return true
		}
	}
	return false
}
