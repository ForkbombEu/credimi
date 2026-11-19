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
	ensureRunnerNamespaceFn      = hooks.EnsureRunnerNamespace
	stopRunnerNamespaceWorkersFn = hooks.StopRunnerNamespaceWorkers
)

// RegisterMobileRunnerNamespaceHooks prepares a runner's namespace while the
// runner is not disabled, and stops the namespace's workers otherwise. Every
// save of a disabled runner clears its namespace readiness. Namespaces are
// never deleted from Temporal; their histories expire by retention.
func RegisterMobileRunnerNamespaceHooks(app core.App) {
	app.OnRecordUpdate("mobile_runners").BindFunc(func(e *core.RecordEvent) error {
		if !mobilerunner.NamespaceEnabled(e.Record) {
			e.Record.Set(mobilerunner.RunnerNamespaceReadyField, false)
		}
		return e.Next()
	})

	app.OnRecordAfterCreateSuccess("mobile_runners").BindFunc(func(e *core.RecordEvent) error {
		if mobilerunner.NamespaceEnabled(e.Record) {
			go ensureRunnerNamespaceFn(e.App, e.Record.Id)
		}
		return e.Next()
	})

	app.OnRecordAfterUpdateSuccess("mobile_runners").BindFunc(func(e *core.RecordEvent) error {
		switch {
		case mobilerunner.NamespaceEnabled(e.Record):
			go ensureRunnerNamespaceFn(e.App, e.Record.Id)
		case mobilerunner.NamespaceEnabled(e.Record.Original()):
			go stopRunnerNamespaceWorkersFn(e.App, e.Record.Id)
		}
		return e.Next()
	})

	app.OnRecordAfterDeleteSuccess("mobile_runners").BindFunc(func(e *core.RecordEvent) error {
		go stopRunnerNamespaceWorkersFn(e.App, e.Record.Id)
		return e.Next()
	})
}
