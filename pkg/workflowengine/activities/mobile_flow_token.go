// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"fmt"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/signedtoken"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/sdk/activity"
)

const (
	// SignMobileFlowTokenActivityName signs the mobile flow token of the
	// calling pipeline run.
	SignMobileFlowTokenActivityName = "Sign a mobile flow token"
	// MobileFlowTokenLabel is the signing label of mobile flow tokens, shared
	// by the activity that signs them and POST /api/pipeline/mobile-flow,
	// which verifies them.
	MobileFlowTokenLabel = "credimi/mobile-flow/v1"
)

// MobileFlowGrant names the pipeline run a mobile flow token was issued to.
type MobileFlowGrant struct {
	Namespace  string `json:"namespace"`
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
}

// SignMobileFlowTokenActivity signs a MobileFlowGrant of the workflow run that
// calls it, never of another: the run's wallet conformance checks pass the
// token to POST /api/pipeline/mobile-flow, which runs wallet actions on that
// run's devices only.
type SignMobileFlowTokenActivity struct {
	workflowengine.BaseActivity
	app core.App
}

// NewSignMobileFlowTokenActivity returns the activity that signs mobile flow
// tokens with app's key.
func NewSignMobileFlowTokenActivity(app core.App) *SignMobileFlowTokenActivity {
	return &SignMobileFlowTokenActivity{
		BaseActivity: workflowengine.BaseActivity{Name: SignMobileFlowTokenActivityName},
		app:          app,
	}
}

// Name is SignMobileFlowTokenActivityName.
func (a *SignMobileFlowTokenActivity) Name() string {
	return a.BaseActivity.Name
}

// Execute signs the calling run's MobileFlowGrant; it ignores its input and
// refuses runs of runner namespaces.
func (a *SignMobileFlowTokenActivity) Execute(
	ctx context.Context,
	_ workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	info := activity.GetInfo(ctx)
	token, err := a.sign(info.Namespace, info.WorkflowExecution.ID, info.WorkflowExecution.RunID)
	if err != nil {
		return result, err
	}
	result.Output = token
	return result, nil
}

// sign signs the MobileFlowGrant of the run runID of workflowID in namespace,
// refusing runner namespaces.
func (a *SignMobileFlowTokenActivity) sign(namespace, workflowID, runID string) (string, error) {
	if strings.HasPrefix(namespace, canonify.RunnerNamespacePrefix) {
		return "", a.NewCodedError(
			errorcodes.MissingOrInvalidPayload,
			false,
			fmt.Errorf("no mobile flow token for runner namespace %s", namespace),
		)
	}
	token, err := signedtoken.Sign(a.app, MobileFlowTokenLabel, MobileFlowGrant{
		Namespace:  namespace,
		WorkflowID: workflowID,
		RunID:      runID,
	})
	if err != nil {
		return "", a.NewCodedError(
			errorcodes.CommandExecutionFailed,
			true,
			fmt.Errorf("sign mobile flow token: %w", err),
		)
	}
	return token, nil
}
