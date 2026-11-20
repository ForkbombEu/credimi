// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package runnerworkflows decides which workflows of a runner namespace an
// organization may see: the device work of its own pipeline runs.
//
// Credimi starts that device work with the memo key
// mobiledevice.PipelineRunMemoKey naming the pipeline run. The memo is
// a claim: anyone holding the runner's token can write it. OwnedBy therefore
// also requires the pipeline run to have used a device of that runner, so only
// the runner that ran a pipeline's devices could forge it, and that runner
// already controls those results.
package runnerworkflows

import (
	"context"
	"database/sql"
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
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevice"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

// ownedTypes are the workflow types Credimi starts in a runner namespace for
// a pipeline run.
var ownedTypes = map[string]bool{
	mobiledevice.PrepareWorkflowName: true,
	mobiledevice.CleanUpWorkflowName: true,
	mobiledevice.StepWorkflowName:    true,
}

// clientFor returns the client of a runner namespace; tests stub it.
var clientFor = temporalclient.GetTemporalClientWithNamespace

// owned and uses cache lookups, because the embedded Temporal UI checks every
// API call of a page. An entry is trusted for cacheTTL only. A cache is
// cleared whole when it passes maxCached entries.
var (
	owned = &lookupCache{}
	uses  = &lookupCache{}
)

const (
	maxCached = 4096
	cacheTTL  = 60 * time.Second
)

// now tells the time of lookups; tests stub it.
var now = time.Now

type ownedKey struct {
	organizationID, namespace, workflowID, runID string
}

type usesKey struct {
	organizationID, runnerID string
}

type cachedLookup struct {
	result   bool
	lookedUp time.Time
}

// lookupCache holds recent lookup results by key.
type lookupCache struct {
	entries sync.Map
	count   atomic.Int64
}

// get returns the result of key, looking it up again once it is older than
// cacheTTL. Failed lookups are not cached.
func (c *lookupCache) get(key any, lookup func() (bool, error)) (bool, error) {
	if cached, ok := c.entries.Load(key); ok {
		if entry, ok := cached.(cachedLookup); ok && now().Sub(entry.lookedUp) < cacheTTL {
			return entry.result, nil
		}
	}
	result, err := lookup()
	if err != nil {
		c.entries.Delete(key)
		return false, err
	}
	if c.count.Add(1) > maxCached {
		c.entries.Clear()
		c.count.Store(1)
	}
	c.entries.Store(key, cachedLookup{result: result, lookedUp: now()})
	return result, nil
}

// clear empties the cache; tests use it.
func (c *lookupCache) clear() {
	c.entries.Clear()
	c.count.Store(0)
}

// OwnedBy reports whether the run runID of workflowID (its latest run when
// runID is empty) in the runner namespace namespace is device work of a
// pipeline run of organizationID:
//   - the namespace's runner exists and is not disabled;
//   - its type is mobiledevice.PrepareWorkflowName,
//     mobiledevice.CleanUpWorkflowName or mobiledevice.StepWorkflowName;
//   - its memo names a pipeline_results row owned by organizationID;
//   - one of that row's devices is hosted by the runner namespace belongs to.
//
// Temporal is asked only for the namespaces of existing runners, so callers
// cannot make Credimi open clients for arbitrary namespaces. A workflow or
// pipeline result that does not exist is not owned.
func OwnedBy(
	ctx context.Context,
	app core.App,
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
	return owned.get(key, func() (bool, error) {
		runner, err := app.FindRecordById("mobile_runners", runnerID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return false, nil
			}
			return false, fmt.Errorf("find runner %s: %w", runnerID, err)
		}
		if runner.GetBool("disabled") {
			return false, nil
		}
		c, err := clientFor(namespace)
		if err != nil {
			return false, fmt.Errorf("temporal client for %s: %w", namespace, err)
		}
		return lookup(ctx, app, c, organizationID, runnerID, workflowID, runID)
	})
}

// UsesRunner reports whether a pipeline run of organizationID used a device
// hosted by the runner with record ID runnerID: a pipeline_results row owned by
// the organization lists such a device.
func UsesRunner(app core.App, organizationID, runnerID string) (bool, error) {
	if organizationID == "" || runnerID == "" {
		return false, nil
	}
	key := usesKey{organizationID: organizationID, runnerID: runnerID}
	return uses.get(key, func() (bool, error) {
		_, err := app.FindFirstRecordByFilter(
			"pipeline_results",
			"owner = {:organization} && devices.runner ?= {:runner}",
			dbx.Params{"organization": organizationID, "runner": runnerID},
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return false, nil
			}
			return false, fmt.Errorf("find pipeline results using runner %s: %w", runnerID, err)
		}
		return true, nil
	})
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
) (mobiledevice.PipelineRun, bool) {
	var pipelineRun mobiledevice.PipelineRun
	payload := fields[mobiledevice.PipelineRunMemoKey]
	if payload == nil ||
		temporalcrypto.DataConverter().FromPayload(payload, &pipelineRun) != nil {
		return pipelineRun, false
	}
	return pipelineRun, pipelineRun.WorkflowID != "" && pipelineRun.RunID != ""
}
