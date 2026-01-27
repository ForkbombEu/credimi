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
	Kind                TaskKind
	NewFunc             func() any
	PayloadType         reflect.Type
	PipelinePayloadType reflect.Type
	OutputKind          workflowengine.OutputKind
	CustomTaskQueue     bool
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
		Kind:                TaskWorkflow,
		NewFunc:             func() any { return workflows.NewMobileAutomationWorkflow() },
		PayloadType:         reflect.TypeOf(workflows.MobileAutomationWorkflowPayload{}),
		CustomTaskQueue: true,
		PipelinePayloadType: reflect.TypeOf(workflows.MobileAutomationWorkflowPipelinePayload{}),
	},
	"custom-check": {
		Kind:        TaskWorkflow,
		NewFunc:     func() any { return workflows.NewCustomCheckWorkflow() },
		PayloadType: reflect.TypeOf(workflows.CustomCheckWorkflowPayload{}),
	},
	"credential-offer": {
		Kind:        TaskWorkflow,
		NewFunc:     func() any { return workflows.NewGetCredentialOfferWorkflow() },
		PayloadType: reflect.TypeOf(workflows.GetCredentialOfferWorkflowPayload{}),
	},
	"conformance-check": {
		Kind:                TaskWorkflow,
		NewFunc:             func() any { return workflows.NewStartCheckWorkflow() },
		PayloadType:         reflect.TypeOf(workflows.StartCheckWorkflowPayload{}),
		PipelinePayloadType: reflect.TypeOf(workflows.StartCheckWorkflowPipelinePayload{}),
	},
	"use-case-verification-deeplink": {
		Kind:        TaskWorkflow,
		NewFunc:     func() any { return workflows.NewGetUseCaseVerificationDeeplinkWorkflow() },
		PayloadType: reflect.TypeOf(workflows.GetUseCaseVerificationDeeplinkWorkflowPayload{}),
	},
}

var PipelineInternalRegistry = map[string]TaskFactory{
	"openidnet-logs": {
		Kind:    TaskWorkflow,
		NewFunc: func() any { return workflows.NewOpenIDNetLogsWorkflow() },
	},
	"ewc-status": {
		Kind:    TaskWorkflow,
		NewFunc: func() any { return workflows.NewEWCStatusWorkflow() },
	},
	"check-file-exists": {
		Kind:       TaskActivity,
		NewFunc:    func() any { return activities.NewCheckFileExistsActivity() },
		OutputKind: workflowengine.OutputBool,
	},
}

// Denylist of task keys that should NOT be registered in the pipeline worker
var PipelineWorkerDenylist = map[string]struct{}{
	"mobile-automation": {},
}
