// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package avdpool

import (
	"errors"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

func StartPoolManagerWorkflow(
	namespace string,
	taskQueue string,
	config PoolConfig,
) (workflowengine.WorkflowResult, error) {
	workflowInput := workflowengine.WorkflowInput{
		Payload: PoolManagerWorkflowInput{
			Config: ApplyPoolConfigDefaults(config),
		},
		Config: map[string]any{},
	}
	workflowOptions := client.StartWorkflowOptions{
		ID:        DefaultPoolWorkflowID,
		TaskQueue: taskQueue,
	}

	w := NewPoolManagerWorkflow()
	result, err := workflowengine.StartWorkflowWithOptions(namespace, workflowOptions, w.Name(), workflowInput)
	if err != nil {
		var alreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &alreadyStarted) {
			return result, nil
		}
		return workflowengine.WorkflowResult{}, fmt.Errorf("failed to start pool manager workflow: %w", err)
	}

	return result, nil
}
