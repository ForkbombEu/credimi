// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"log"

	"github.com/forkbombeu/credimi/pkg/workflowengine/hooks"
	"github.com/pocketbase/pocketbase/core"
)

var (
	ensureNamespaceReadyFn      = hooks.EnsureNamespaceReady
	startWorkersByNamespaceFn   = hooks.StartAllWorkersByNamespace
	ensureNamespaceAndWorkersFn = ensureNamespaceAndWorkers
)

const defaultMaxPipelinesInQueue = 1

func HookOrganizations(app core.App) {
	registerOrganizationNamespaceHooks(app)
	registerOrganizationProtectedFieldsHooks(app)
}

func registerOrganizationNamespaceHooks(app core.App) {
	app.OnRecordCreate("organizations").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetInt("max_pipelines_in_queue") == 0 {
			e.Record.Set("max_pipelines_in_queue", defaultMaxPipelinesInQueue)
		}
		return e.Next()
	})

	app.OnRecordAfterCreateSuccess("organizations").BindFunc(func(e *core.RecordEvent) error {
		orgName := e.Record.GetString("canonified_name")
		if orgName != "" {
			ensureNamespaceAndWorkersFn(e.App, orgName)
		}

		return e.Next()
	})

	app.OnRecordAfterUpdateSuccess("organizations").BindFunc(func(e *core.RecordEvent) error {
		oldName := e.Record.Original().GetString("canonified_name")
		newName := e.Record.GetString("canonified_name")

		if oldName == newName || newName == "" {
			return e.Next()
		}

		go hooks.StopAllWorkersByNamespace(oldName)

		ensureNamespaceAndWorkersFn(e.App, newName)
		log.Printf("Moved workers from namespace %s to %s", oldName, newName)
		return e.Next()
	})
}

func registerOrganizationProtectedFieldsHooks(app core.App) {
	app.OnRecordUpdateRequest("organizations").BindFunc(func(e *core.RecordRequestEvent) error {
		if e.HasSuperuserAuth() {
			return e.Next()
		}

		original := e.Record.Original()
		if original == nil {
			return e.Next()
		}

		if e.Record.GetInt("max_pipelines_in_queue") != original.GetInt("max_pipelines_in_queue") {
			e.Record.Set("max_pipelines_in_queue", original.GetInt("max_pipelines_in_queue"))
		}

		return e.Next()
	})
}

// ensureNamespaceAndWorkers ensures the given namespace exists in Temporal,
// creating it and its search attributes when missing, then starts all workers
// for that namespace in a goroutine.
func ensureNamespaceAndWorkers(app core.App, namespace string) {
	if hooks.TemporalWorkersDisabled() {
		log.Printf(
			"Skipping namespace %s (%s is set)",
			namespace,
			hooks.TemporalWorkersDisabledEnv,
		)
		return
	}

	if err := ensureNamespaceReadyFn(namespace); err != nil {
		log.Printf("Namespace %s not ready after retries: %v", namespace, err)
		return
	}

	go startWorkersByNamespaceFn(app, namespace)
}
