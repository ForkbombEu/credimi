// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package runnerworkflows decides which workflows of a runner namespace an
// organization may see: the device work of its own pipeline runs.
//
// Credimi starts that device work with the memo key
// mobiledevicesession.PipelineRunMemoKey naming the pipeline run. The memo is
// a claim: anyone holding the runner's token can write it. OwnedBy therefore
// also requires the pipeline run to have used a device of that runner, so only
// the runner that ran a pipeline's devices could forge it, and that runner
// already controls those results.
package runnerworkflows

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	pipelineresults "github.com/forkbombeu/credimi/pkg/internal/pipeline_results"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesession"
	"github.com/pocketbase/pocketbase/core"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

// ownedTypes are the workflow types Credimi starts in a runner namespace for
// a pipeline run.
var ownedTypes = map[string]bool{
	mobiledevicesession.WorkflowName:                 true,
	mobiledevicesession.MobileAutomationWorkflowName: true,
}

// owned caches lookups, because the embedded Temporal UI checks every API call
// of a page. An entry is trusted for ownedTTL only. It is cleared whole when it
// passes maxOwned entries.
var (
	owned      sync.Map
	ownedCount atomic.Int64
)

const (
	maxOwned = 4096
	ownedTTL = 60 * time.Second
)

// now tells the time of lookups; tests stub it.
var now = time.Now

type ownedKey struct {
	organizationID, namespace, workflowID, runID string
}

type ownedEntry struct {
	owned    bool
	lookedUp time.Time
}

// OwnedBy reports whether the run runID of workflowID (its latest run when
// runID is empty) in the runner namespace namespace is device work of a
// pipeline run of organizationID:
//   - its type is mobiledevicesession.WorkflowName or
//     mobiledevicesession.MobileAutomationWorkflowName;
//   - its memo names a pipeline_results row owned by organizationID;
//   - one of that row's devices is hosted by the runner namespace belongs to.
//
// c must be a client of namespace. A workflow or pipeline result that does not
// exist is not owned.
func OwnedBy(
	ctx context.Context,
	app core.App,
	c client.Client,
	organizationID, namespace, workflowID, runID string,
) (bool, error) {
	runnerID, ok := strings.CutPrefix(namespace, canonify.RunnerNamespacePrefix)
	if !ok || runnerID == "" || organizationID == "" || workflowID == "" {
		return false, nil
	}
	key := ownedKey{
		organizationID: organizationID,
		namespace:      namespace,
		workflowID:     workflowID,
		runID:          runID,
	}
	if cached, ok := owned.Load(key); ok {
		if entry, ok := cached.(ownedEntry); ok && now().Sub(entry.lookedUp) < ownedTTL {
			return entry.owned, nil
		}
	}
	result, err := lookup(ctx, app, c, organizationID, runnerID, workflowID, runID)
	if err != nil {
		owned.Delete(key)
		return false, err
	}
	if ownedCount.Add(1) > maxOwned {
		owned.Clear()
		ownedCount.Store(1)
	}
	owned.Store(key, ownedEntry{owned: result, lookedUp: now()})
	return result, nil
}

func lookup(
	ctx context.Context,
	app core.App,
	c client.Client,
	organizationID, runnerID, workflowID, runID string,
) (bool, error) {
	description, err := temporalclient.Describe(ctx, c, workflowID, runID)
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, fmt.Errorf("describe workflow %s: %w", workflowID, err)
	}
	info := description.GetWorkflowExecutionInfo()
	if !ownedTypes[info.GetType().GetName()] {
		return false, nil
	}
	pipelineRun, ok := memoPipelineRun(info.GetMemo().GetFields())
	if !ok {
		return false, nil
	}
	result, err := pipelineresults.FindByWorkflowRun(
		app,
		pipelineRun.WorkflowID,
		pipelineRun.RunID,
	)
	if err != nil {
		if errors.Is(err, pipelineresults.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	if result.GetString("owner") != organizationID {
		return false, nil
	}
	deviceIDs := result.GetStringSlice("devices")
	if len(deviceIDs) == 0 {
		return false, nil
	}
	devices, err := app.FindRecordsByIds("mobile_devices", deviceIDs)
	if err != nil {
		return false, fmt.Errorf("find devices of pipeline result %s: %w", result.Id, err)
	}
	for _, device := range devices {
		if device.GetString("runner") == runnerID {
			return true, nil
		}
	}
	return false, nil
}

// memoPipelineRun decodes the pipeline run a workflow's memo claims; a
// missing or malformed claim owns nothing.
func memoPipelineRun(
	fields map[string]*commonpb.Payload,
) (mobiledevicesession.PipelineRunRef, bool) {
	var pipelineRun mobiledevicesession.PipelineRunRef
	payload := fields[mobiledevicesession.PipelineRunMemoKey]
	if payload == nil ||
		temporalcrypto.DataConverter().FromPayload(payload, &pipelineRun) != nil {
		return pipelineRun, false
	}
	return pipelineRun, pipelineRun.WorkflowID != "" && pipelineRun.RunID != ""
}
