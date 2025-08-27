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
	"http": {
		NewFunc:    func() workflowengine.Activity { return NewHTTPActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"docker": {
		NewFunc:    func() workflowengine.Activity { return NewDockerActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"mail": {
		NewFunc:    func() workflowengine.Activity { return NewSendMailActivity() },
		OutputKind: workflowengine.OutputString,
	},
	"stepCI": {
		NewFunc:    func() workflowengine.Activity { return NewStepCIWorkflowActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"json": {
		NewFunc: func() workflowengine.Activity {
			return NewJSONActivity(map[string]reflect.Type{
				"map": reflect.TypeOf(
					map[string]any{},
				),
			})
		},
		OutputKind: workflowengine.OutputMap,
	},
	"validateSchema": {
		NewFunc:    func() workflowengine.Activity { return NewSchemaValidationActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"parseCredentialsIssuer": {
		NewFunc:    func() workflowengine.Activity { return NewCheckCredentialsIssuerActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"parseCESR": {
		NewFunc:    func() workflowengine.Activity { return NewCESRParsingActivity() },
		OutputKind: workflowengine.OutputMap,
	},
	"validateCESR": {
		NewFunc:    func() workflowengine.Activity { return NewCESRValidateActivity() },
		OutputKind: workflowengine.OutputAny,
	},
	"parseWalletURL": {
		NewFunc:    func() workflowengine.Activity { return NewParseWalletURLActivity() },
		OutputKind: workflowengine.OutputMap,
	},
}
