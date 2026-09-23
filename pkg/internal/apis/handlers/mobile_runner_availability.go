// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
)

// Catalog surfaces report availability from heartbeat freshness, which is
// necessarily a moment old by the time an operator picks a device. Every path
// that is about to commit a run resolves availability here instead, against the
// runner itself.

const runnerHealthTimeout = 10 * time.Second

var checkRunnerReachable = checkRunnerReachableHTTP

// requireMobileDeviceRunnersOnline rejects a run unless every chosen device is
// hosted by a runner answering its health endpoint. Runners are probed once
// even when a run spans several of their devices: a pipeline that needs three
// devices and gets two fails anyway, so partial availability is not a run.
func requireMobileDeviceRunnersOnline(
	ctx context.Context,
	app core.App,
	deviceIDs []string,
) *apierror.APIError {
	probed := make(map[string]struct{}, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		runner, apiErr := mobileDeviceRunnerRecord(app, deviceID)
		if apiErr != nil {
			return apiErr
		}
		if _, done := probed[runner.Id]; done {
			continue
		}
		probed[runner.Id] = struct{}{}
		online, apiErr := mobileRunnerReachable(ctx, runner)
		if apiErr != nil {
			return apiErr
		}
		if !online {
			return apierror.New(
				http.StatusServiceUnavailable,
				"device_id",
				"device runner is offline",
				"mobile device "+deviceID+" is hosted by a runner that is not reachable",
			)
		}
	}

	return nil
}

func requireMobileDeviceRunnerOnline(
	ctx context.Context,
	app core.App,
	deviceID string,
) *apierror.APIError {
	return requireMobileDeviceRunnersOnline(ctx, app, []string{deviceID})
}

func mobileDeviceRunnerRecord(app core.App, deviceID string) (*core.Record, *apierror.APIError) {
	record, err := canonify.Resolve(app, canonify.NormalizePath(deviceID))
	if err != nil || record == nil || record.Collection() == nil ||
		record.Collection().Name != mobileDevicesCollection {
		return nil, apierror.New(
			http.StatusNotFound,
			"device_id",
			"mobile_device_not_found",
			"mobile device "+deviceID+" not found",
		)
	}
	runner, err := app.FindRecordById("mobile_runners", record.GetString("runner"))
	if err != nil {
		return nil, apierror.New(
			http.StatusNotFound,
			"device_id",
			"mobile_runner_not_found",
			"mobile device runner not found",
		)
	}

	return runner, nil
}

func mobileRunnerReachable(ctx context.Context, record *core.Record) (bool, *apierror.APIError) {
	runnerURL := mobileRunnerURL(record)
	if runnerURL == "" {
		return false, nil
	}

	online, err := checkRunnerReachable(ctx, runnerURL)
	if err != nil {
		return false, apierror.New(
			http.StatusInternalServerError,
			"device_type",
			"failed to check runner health",
			err.Error(),
		)
	}

	return online, nil
}

func checkRunnerReachableHTTP(ctx context.Context, runnerURL string) (bool, error) {
	healthURL, err := url.JoinPath(runnerURL, "health")
	if err != nil {
		return false, err
	}

	healthCtx, cancel := context.WithTimeout(ctx, runnerHealthTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(healthCtx, http.MethodGet, healthURL, nil)
	if err != nil {
		return false, err
	}

	resp, err := mobileRunnerHTTPClient(runnerURL).Do(req)
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}
