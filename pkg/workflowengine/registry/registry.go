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
	Kind        TaskKind
	NewFunc     func() any
	PayloadType reflect.Type
	OutputKind  workflowengine.OutputKind
	TaskQueue   string
}

// Registry maps activity keys to their factory.
var Registry = map[string]TaskFactory{
	"http-request": {
		Kind:        TaskActivity,
		NewFunc:     func() any { return activities.NewHTTPActivity() },
		PayloadType: reflect.TypeOf(activities.HTTPActivityPayload{}),
		OutputKind:  workflowengine.OutputMap,
	},
	"container-run": {
		Kind:        TaskActivity,
		NewFunc:     func() any { return activities.NewDockerActivity() },
		PayloadType: reflect.TypeOf(activities.DockerActivityPayload{}),
		OutputKind:  workflowengine.OutputMap,
	},
	"email": {
		Kind:        TaskActivity,
		NewFunc:     func() any { return activities.NewSendMailActivity() },
		PayloadType: reflect.TypeOf(activities.SendMailActivityPayload{}),
		OutputKind:  workflowengine.OutputString,
	},
	"rest-chain": {
		Kind:        TaskActivity,
		NewFunc:     func() any { return activities.NewStepCIWorkflowActivity() },
		PayloadType: reflect.TypeOf(activities.StepCIWorkflowActivityPayload{}),
		OutputKind:  workflowengine.OutputMap,
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
		PayloadType: reflect.TypeOf(activities.JSONActivityPayload{}),
		OutputKind:  workflowengine.OutputMap,
	},
	"jsonschema-validation": {
		Kind:        TaskActivity,
		NewFunc:     func() any { return activities.NewSchemaValidationActivity() },
		PayloadType: reflect.TypeOf(activities.SchemaValidationActivityPayload{}),
		OutputKind:  workflowengine.OutputMap,
	},
	"credential-issuer-validation": {
		Kind:        TaskActivity,
		NewFunc:     func() any { return activities.NewCheckCredentialsIssuerActivity() },
		PayloadType: reflect.TypeOf(activities.CheckCredentialsIssuerActivityPayload{}),
		OutputKind:  workflowengine.OutputMap,
	},
	"cesr-parse": {
		Kind:        TaskActivity,
		NewFunc:     func() any { return activities.NewCESRParsingActivity() },
		PayloadType: reflect.TypeOf(activities.CESRParsingActivityPayload{}),
		OutputKind:  workflowengine.OutputMap,
	},
	"cesr-validate": {
		Kind:        TaskActivity,
		NewFunc:     func() any { return activities.NewCESRValidateActivity() },
		PayloadType: reflect.TypeOf(activities.CesrValidateActivityPayload{}),
		OutputKind:  workflowengine.OutputAny,
	},
	"appstore-url-validation": {
		Kind:        TaskActivity,
		NewFunc:     func() any { return activities.NewParseWalletURLActivity() },
		PayloadType: reflect.TypeOf(activities.ParseWalletURLActivityPayload{}),
		OutputKind:  workflowengine.OutputMap,
	},
	"mobile-automation": {
		Kind:        TaskWorkflow,
		NewFunc:     func() any { return &workflows.MobileAutomationWorkflow{} },
		PayloadType: reflect.TypeOf(workflows.MobileAutomationWorkflowPayload{}),
		TaskQueue:   workflows.MobileAutomationTaskQueue,
	},
	"custom-check": {
		Kind:        TaskWorkflow,
		NewFunc:     func() any { return &workflows.CustomCheckWorkflow{} },
		PayloadType: reflect.TypeOf(workflows.CustomCheckWorkflowPayload{}),
	},
	"credential-offer": {
		Kind:        TaskWorkflow,
		NewFunc:     func() any { return &workflows.GetCredentialOfferWorkflow{} },
		PayloadType: reflect.TypeOf(workflows.GetCredentialOfferWorkflowPayload{}),
	},
	"conformance-check": {
		Kind:        TaskWorkflow,
		NewFunc:     func() any { return &workflows.StartCheckWorkflow{} },
		PayloadType: reflect.TypeOf(workflows.StartCheckWorkflowPayload{}),
	},
	"use-case-verification-deeplink": {
		Kind:        TaskWorkflow,
		NewFunc:     func() any { return &workflows.GetUseCaseVerificationDeeplinkWorkflow{} },
		PayloadType: reflect.TypeOf(workflows.GetUseCaseVerificationDeeplinkWorkflowPayload{}),
	},
}

var PipelineInternalRegistry = map[string]TaskFactory{
	"openidnet-logs": {
		Kind:    TaskWorkflow,
		NewFunc: func() any { return &workflows.OpenIDNetLogsWorkflow{} },
	},
	"ewc-status": {
		Kind:    TaskWorkflow,
		NewFunc: func() any { return &workflows.EWCStatusWorkflow{} },
	},
	"check-file-exists": {
		Kind:       TaskActivity,
		NewFunc:    func() any { return activities.NewCheckFileExistsActivity() },
		OutputKind: workflowengine.OutputBool,
	},
}

// Denylist of task keys that should NOT be registered in the pipeline worker
var PipelineWorkerDenylist = map[string]struct{}{
	"mobile-flow":       {},
	"mobile-automation": {},
	"apk-install":       {},
	"apk-uninstall":     {},
}
