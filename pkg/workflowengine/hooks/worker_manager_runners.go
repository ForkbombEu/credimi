// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package hooks

import (
	"log"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunnerlifecycle"
	"github.com/pocketbase/pocketbase/core"
)

func WorkerManagerAdminRunnerIDs(app core.App) ([]string, error) {
	return listWorkerManagerRunnerIDs(app, "admin_managed = true")
}

func WorkerManagerPublishedNonAdminRunnerIDs(app core.App) ([]string, error) {
	return listWorkerManagerRunnerIDs(app, "published = true && admin_managed = false")
}

func workerManagerAllOrganizationRecords(app core.App) ([]*core.Record, error) {
	return app.FindRecordsByFilter("organizations", "", "name", -1, 0)
}

// listWorkerManagerRunnerIDs returns the identifiers of the startable runners
// matching filter that have an address to be called at. A runner whose
// identifier cannot be built (for example, its owner organization is gone) is
// skipped and logged: one broken record must not stop server startup or fail
// an organization save.
func listWorkerManagerRunnerIDs(app core.App, filter string) ([]string, error) {
	records, err := app.FindRecordsByFilter("mobile_runners", filter, "name", -1, 0)
	if err != nil {
		return nil, err
	}

	runnerIDs := make([]string, 0, len(records))
	for _, record := range records {
		if !mobilerunnerlifecycle.EligibleForWorkerStart(record) {
			continue
		}
		if mobilerunner.RunnerURL(record) == "" {
			continue
		}
		runnerID, err := mobilerunner.RunnerIdentifier(app, record)
		if err != nil {
			log.Printf("[WorkersHook] Skipping mobile runner %s: %v", record.Id, err)
			continue
		}
		runnerIDs = append(runnerIDs, runnerID)
	}

	return uniqueWorkerManagerRunnerIDs(runnerIDs), nil
}

func uniqueWorkerManagerRunnerIDs(runnerIDs []string) []string {
	seen := make(map[string]struct{}, len(runnerIDs))
	result := make([]string, 0, len(runnerIDs))
	for _, runnerID := range runnerIDs {
		trimmed := strings.TrimSpace(runnerID)
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

func combineWorkerManagerRunnerIDs(groups ...[]string) []string {
	combined := make([]string, 0)
	for _, group := range groups {
		combined = append(combined, group...)
	}

	return uniqueWorkerManagerRunnerIDs(combined)
}
