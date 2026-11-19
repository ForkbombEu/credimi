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
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"go.temporal.io/sdk/activity"
)

// callerRunnerNamespace returns the namespace of the workflow that scheduled
// the activity when it is a runner namespace, and "" otherwise.
func callerRunnerNamespace(ctx context.Context) string {
	if !activity.IsActivity(ctx) {
		return ""
	}
	namespace := activity.GetInfo(ctx).WorkflowNamespace
	if !strings.HasPrefix(namespace, canonify.RunnerNamespacePrefix) {
		return ""
	}
	return namespace
}

// requireRunnerNamespaceOfCaller lets an activity scheduled from a runner
// namespace act only on that namespace's runner, whose namespace is
// runnerNamespace: the runner owner holds write access to its own namespace
// and could otherwise reach other runners' devices through Credimi. Calls
// from organization namespaces and default pass, since only Credimi writes
// there.
func requireRunnerNamespaceOfCaller(
	ctx context.Context,
	a *workflowengine.BaseActivity,
	runnerNamespace string,
) error {
	return checkRunnerNamespaceOfCaller(a, callerRunnerNamespace(ctx), runnerNamespace)
}

// checkRunnerNamespaceOfCaller is requireRunnerNamespaceOfCaller for the
// caller's runner namespace caller, "" for any other namespace.
func checkRunnerNamespaceOfCaller(
	a *workflowengine.BaseActivity,
	caller, runnerNamespace string,
) error {
	if caller == "" || caller == runnerNamespace {
		return nil
	}
	return a.NewCodedError(
		errorcodes.MissingOrInvalidPayload,
		false,
		fmt.Errorf(
			"activity called from %s for runner %s",
			caller,
			strings.TrimPrefix(runnerNamespace, canonify.RunnerNamespacePrefix),
		),
	)
}
