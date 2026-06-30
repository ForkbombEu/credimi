// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

func RegisterMobileRunnerWorkerManagerHooks(app core.App) {
	app.OnRecordAfterUpdateSuccess("mobile_runners").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetBool("admin_managed") {
			return e.Next()
		}
		if e.Record.Original().GetBool("published") || !e.Record.GetBool("published") {
			return e.Next()
		}

		runnerURL := mobileRunnerURL(e.Record)
		if runnerURL == "" {
			return e.Next()
		}

		orgs, err := listPublishedOrganizationRecords(e.App)
		if err != nil {
			return err
		}

		for _, org := range orgs {
			namespace := org.GetString("canonified_name")
			if namespace == "" {
				continue
			}
			startWorkerManagerFn(e.App, namespace, "", []string{runnerURL})
		}

		return e.Next()
	})
}

func listPublishedOrganizationRecords(app core.App) ([]*core.Record, error) {
	return app.FindRecordsByFilter("organizations", "published = true", "name", -1, 0)
}

func mobileRunnerURL(record *core.Record) string {
	runnerURL := strings.TrimSpace(record.GetString("ip"))
	if runnerURL == "" {
		return ""
	}

	if port := strings.TrimSpace(record.GetString("port")); port != "" {
		runnerURL = strings.TrimRight(runnerURL, "/") + ":" + port
	}

	return runnerURL
}
