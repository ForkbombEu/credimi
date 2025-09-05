// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"reflect"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

// ActivityFactory holds the constructor and output kind for a registered activity.
type ActivityFactory struct {
	NewFunc    func() workflowengine.Activity
	OutputKind workflowengine.OutputKind
}

// Registry maps activity keys to their factory.
var Registry = map[string]ActivityFactory{
	"http-request": {
		NewFunc:    func() workflowengine.Activity { return NewHTTPActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"container-run": {
		NewFunc:    func() workflowengine.Activity { return NewDockerActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"email": {
		NewFunc:    func() workflowengine.Activity { return NewSendMailActivity() },
		OutputKind: workflowengine.OutputString,
	},
	"rest-chain": {
		NewFunc:    func() workflowengine.Activity { return NewStepCIWorkflowActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"json-parse": {
		NewFunc: func() workflowengine.Activity {
			return NewJSONActivity(map[string]reflect.Type{
				"map": reflect.TypeOf(
					map[string]any{},
				),
			})
		},
		OutputKind: workflowengine.OutputMap,
	},
	"jsonschema-validation": {
		NewFunc:    func() workflowengine.Activity { return NewSchemaValidationActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"credential-issuer-validation": {
		NewFunc:    func() workflowengine.Activity { return NewCheckCredentialsIssuerActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"cesr-parse": {
		NewFunc:    func() workflowengine.Activity { return NewCESRParsingActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"cesr-validate": {
		NewFunc:    func() workflowengine.Activity { return NewCESRValidateActivity() },
		OutputKind: workflowengine.OutputAny,
	},
	"appstore-url-validation": {
		NewFunc:    func() workflowengine.Activity { return NewParseWalletURLActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"maestro-flow": {
		NewFunc:    func() workflowengine.Activity { return NewMaestroFlowActivity() },
		OutputKind: workflowengine.OutputString,
	},
}
