// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"fmt"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

var mobileRunnerShutdownTemporalClient = temporalclient.GetTemporalClientWithNamespace

const mobileRunnerShutdownAcceptedTimeout = 5 * time.Second

func RegisterMobileRunnerHooks(app core.App) {
	bindMobileRunnerLifecycleMonitor(app)

	app.OnRecordDelete("mobile_runners").BindFunc(func(e *core.RecordEvent) error {
		deviceRecords, err := app.FindRecordsByFilter(
			"mobile_devices",
			"runner = {:runner}",
			"",
			-1,
			0,
			dbx.Params{"runner": e.Record.Id},
		)
		if err != nil {
			return fmt.Errorf("list mobile runner devices: %w", err)
		}
		for _, device := range deviceRecords {
			if err := shutdownMobileDeviceSemaphore(
				app,
				device,
				"mobile runner deleted",
			); err != nil {
				return err
			}
		}

		return e.Next()
	})
}

func mobileRunnerRecordIdentifier(app core.App, record *core.Record) (string, error) {
	runnerID, err := canonify.BuildPath(
		app,
		record,
		canonify.CanonifyPaths["mobile_runners"],
		"",
	)
	if err != nil {
		return "", err
	}
	return canonify.NormalizePath(runnerID), nil
}
