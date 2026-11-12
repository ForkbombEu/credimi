// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/workflowengine/hooks"
	"github.com/pocketbase/pocketbase/core"
)

var (
	ensureRunnerAgentNamespaceFn = hooks.EnsureRunnerAgentNamespace
	stopRunnerAgentWorkersFn     = hooks.StopRunnerAgentWorkers
)

// RegisterMobileRunnerAgentNamespaceHooks prepares a runner's agent namespace
// while its agent is enabled and the runner is not disabled, and stops the
// namespace's workers otherwise. Every save of a runner whose agent is off or
// that is disabled clears its agent namespace readiness. Namespaces are never
// deleted from Temporal; their histories expire by retention.
func RegisterMobileRunnerAgentNamespaceHooks(app core.App) {
	app.OnRecordUpdate("mobile_runners").BindFunc(func(e *core.RecordEvent) error {
		if !mobilerunner.AgentEnabled(e.Record) {
			e.Record.Set(mobilerunner.AgentNamespaceReadyField, false)
		}
		return e.Next()
	})

	app.OnRecordAfterCreateSuccess("mobile_runners").BindFunc(func(e *core.RecordEvent) error {
		if mobilerunner.AgentEnabled(e.Record) {
			go ensureRunnerAgentNamespaceFn(e.App, e.Record.Id)
		}
		return e.Next()
	})

	app.OnRecordAfterUpdateSuccess("mobile_runners").BindFunc(func(e *core.RecordEvent) error {
		switch {
		case mobilerunner.AgentEnabled(e.Record):
			go ensureRunnerAgentNamespaceFn(e.App, e.Record.Id)
		case mobilerunner.AgentEnabled(e.Record.Original()):
			go stopRunnerAgentWorkersFn(e.App, e.Record.Id)
		}
		return e.Next()
	})

	app.OnRecordAfterDeleteSuccess("mobile_runners").BindFunc(func(e *core.RecordEvent) error {
		go stopRunnerAgentWorkersFn(e.App, e.Record.Id)
		return e.Next()
	})
}
