// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/mobilerunnerlifecycle"
	"github.com/pocketbase/pocketbase/core"
)

// workerManagerDefaultNamespace mirrors the namespace the startup hook always
// attaches admin-managed runners to.
const workerManagerDefaultNamespace = "default"

// RegisterMobileRunnerWorkerManagerHooks starts worker managers for a mobile
// runner when a record update is what made that runner able to take workers.
func RegisterMobileRunnerWorkerManagerHooks(app core.App) {
	app.OnRecordAfterUpdateSuccess("mobile_runners").BindFunc(func(e *core.RecordEvent) error {
		startable := workerManagerRunnerStartable
		listOrganizations := listPublishedOrganizationRecords
		namespaces := []string{}
		if e.Record.GetBool("admin_managed") {
			// Admin-managed runners serve every namespace, so their startable
			// set does not depend on runner publication and always includes the
			// default namespace, exactly like the startup hook.
			startable = workerManagerAdminRunnerStartable
			listOrganizations = listAllOrganizationRecords
			namespaces = append(namespaces, workerManagerDefaultNamespace)
		}

		// Start only when this update is what made the runner startable.
		// Runners that still cannot take workers get nothing, and runners that
		// were already startable are not restarted on unrelated field writes.
		if !startable(e.Record) || startable(e.Record.Original()) {
			return e.Next()
		}

		runnerURL := mobileRunnerURL(e.Record)
		if runnerURL == "" {
			return e.Next()
		}

		orgs, err := listOrganizations(e.App)
		if err != nil {
			return err
		}

		for _, org := range orgs {
			namespace := org.GetString("canonified_name")
			if namespace == "" {
				continue
			}
			namespaces = append(namespaces, namespace)
		}

		for _, namespace := range namespaces {
			startWorkerManagerFn(e.App, namespace, "", []string{runnerURL})
		}

		return e.Next()
	})
}

func listPublishedOrganizationRecords(app core.App) ([]*core.Record, error) {
	return app.FindRecordsByFilter("organizations", "published = true", "name", -1, 0)
}

func listAllOrganizationRecords(app core.App) ([]*core.Record, error) {
	return app.FindRecordsByFilter("organizations", "", "name", -1, 0)
}

// workerManagerRunnerStartable reports whether a non-admin runner record can
// take worker-manager starts for published organizations.
func workerManagerRunnerStartable(record *core.Record) bool {
	if record == nil {
		return false
	}

	return record.GetBool("published") && mobilerunnerlifecycle.EligibleForWorkerStart(record)
}

// workerManagerAdminRunnerStartable reports whether an admin-managed runner
// record can take worker-manager starts. Publication is irrelevant for
// admin-managed runners: they serve every namespace.
func workerManagerAdminRunnerStartable(record *core.Record) bool {
	return mobilerunnerlifecycle.EligibleForWorkerStart(record)
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
