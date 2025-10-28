// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package registry

import (
	"reflect"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
)

type TaskKind int

const (
	TaskActivity TaskKind = iota
	TaskWorkflow
)

type TaskFactory struct {
	Kind       TaskKind
	NewFunc    func() any
	OutputKind workflowengine.OutputKind
	TaskQueue  string
}

// Registry maps activity keys to their factory.
var Registry = map[string]TaskFactory{
	"http-request": {
		Kind:       TaskActivity,
		NewFunc:    func() any { return activities.NewHTTPActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"container-run": {
		Kind:       TaskActivity,
		NewFunc:    func() any { return activities.NewDockerActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"email": {
		Kind:       TaskActivity,
		NewFunc:    func() any { return activities.NewSendMailActivity() },
		OutputKind: workflowengine.OutputString,
	},
	"rest-chain": {
		Kind:       TaskActivity,
		NewFunc:    func() any { return activities.NewStepCIWorkflowActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"json-parse": {
		Kind: TaskActivity,
		NewFunc: func() any {
			return activities.NewJSONActivity(map[string]reflect.Type{
				"map": reflect.TypeOf(
					map[string]any{},
				),
			})
		},
		OutputKind: workflowengine.OutputMap,
	},
	"jsonschema-validation": {
		Kind:       TaskActivity,
		NewFunc:    func() any { return activities.NewSchemaValidationActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"credential-issuer-validation": {
		Kind:       TaskActivity,
		NewFunc:    func() any { return activities.NewCheckCredentialsIssuerActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"cesr-parse": {
		Kind:       TaskActivity,
		NewFunc:    func() any { return activities.NewCESRParsingActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"cesr-validate": {
		Kind:       TaskActivity,
		NewFunc:    func() any { return activities.NewCESRValidateActivity() },
		OutputKind: workflowengine.OutputAny,
	},
	"appstore-url-validation": {
		Kind:       TaskActivity,
		NewFunc:    func() any { return activities.NewParseWalletURLActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"mobile-flow": {
		Kind:       TaskActivity,
		NewFunc:    func() any { return activities.NewRunMobileFlowActivity() },
		OutputKind: workflowengine.OutputString,
	},
	"apk-install": {
		Kind:       TaskActivity,
		NewFunc:    func() any { return activities.NewApkInstallActivity() },
		OutputKind: workflowengine.OutputString,
	},
	"apk-uninstall": {
		Kind:       TaskActivity,
		NewFunc:    func() any { return activities.NewApkUninstallActivity() },
		OutputKind: workflowengine.OutputString,
	},

	"mobile-automation": {
		Kind:      TaskWorkflow,
		NewFunc:   func() any { return &workflows.MobileAutomationWorkflow{} },
		TaskQueue: workflows.MobileAutomationTaskQueue,
	},
	"custom-check": {
		Kind:    TaskWorkflow,
		NewFunc: func() any { return &workflows.CustomCheckWorkflow{} },
	},
	"check-file-exists": {
		Kind:       TaskActivity,
		NewFunc:    func() any { return activities.NewCheckFileExistsActivity() },
		OutputKind: workflowengine.OutputBool,
	},
	"openid-net": {
		Kind:      TaskWorkflow,
		NewFunc:   func() any { return &workflows.OpenIDNetWorkflow{} },
		TaskQueue: workflows.OpenIDNetTaskQueue,
	},
	"credential-offer": {
		Kind:    TaskWorkflow,
		NewFunc: func() any { return &workflows.GetCredentialOfferWorkflow{} },
	},
	"conformance-check": {
		Kind:    TaskWorkflow,
		NewFunc: func() any { return &workflows.StartCheckWorkflow{} },
	},
	"openidnet-logs": {
		Kind:    TaskWorkflow,
		NewFunc: func() any { return &workflows.OpenIDNetLogsWorkflow{} },
	},
	"ewc-status": {
		Kind:    TaskWorkflow,
		NewFunc: func() any { return &workflows.EWCStatusWorkflow{} },
	},
	"use-case-verification-deeplink": {
		Kind:    TaskWorkflow,
		NewFunc: func() any { return &workflows.GetUseCaseVerificationDeeplinkWorkflow{} },
	},
}

// Denylist of task keys that should NOT be registered in the pipeline worker
var PipelineWorkerDenylist = map[string]struct{}{
	"mobile-flow":       {},
	"mobile-automation": {},
	"apk-install":       {},
	"apk-uninstall":     {},
}
