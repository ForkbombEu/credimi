// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package hooks

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

func WorkerManagerAdminRunnerURLs(app core.App) ([]string, error) {
	return listWorkerManagerRunnerURLs(app, "admin_managed = true")
}

func WorkerManagerPublishedNonAdminRunnerURLs(app core.App) ([]string, error) {
	return listWorkerManagerRunnerURLs(app, "published = true && admin_managed = false")
}

func workerManagerAllOrganizationRecords(app core.App) ([]*core.Record, error) {
	return app.FindRecordsByFilter("organizations", "", "name", -1, 0)
}

func listWorkerManagerRunnerURLs(app core.App, filter string) ([]string, error) {
	records, err := app.FindRecordsByFilter("mobile_runners", filter, "name", -1, 0)
	if err != nil {
		return nil, err
	}

	runnerURLs := make([]string, 0, len(records))
	for _, record := range records {
		runnerURL := strings.TrimSpace(record.GetString("ip"))
		if runnerURL == "" {
			continue
		}
		if port := strings.TrimSpace(record.GetString("port")); port != "" {
			runnerURL = strings.TrimRight(runnerURL, "/") + ":" + port
		}
		runnerURLs = append(runnerURLs, runnerURL)
	}

	return uniqueWorkerManagerURLs(runnerURLs), nil
}

func uniqueWorkerManagerURLs(runnerURLs []string) []string {
	seen := make(map[string]struct{}, len(runnerURLs))
	result := make([]string, 0, len(runnerURLs))
	for _, runnerURL := range runnerURLs {
		trimmed := strings.TrimSpace(runnerURL)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}

		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}

	return result
}

func combineWorkerManagerRunnerURLs(groups ...[]string) []string {
	combined := make([]string, 0)
	for _, group := range groups {
		combined = append(combined, group...)
	}

	return uniqueWorkerManagerURLs(combined)
}
