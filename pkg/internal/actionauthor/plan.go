// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"errors"
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

const planKeyLabel = "credimi/action-author/plan/v1"

// PlanGrant is a human's confirmation of the plan of one ActionAuthor
// workflow, signed by Credimi.
type PlanGrant struct {
	AuthorWorkflowID   string `json:"author_workflow_id"`
	Plan               Spec   `json:"plan"`
	ApproverID         string `json:"approver_id"`
	ApproverCollection string `json:"approver_collection"`
	ConfirmedAt        int64  `json:"confirmed_at"`
}

// SignPlan returns the plan token, in the format of SignApproval.
func SignPlan(app core.App, g PlanGrant) (string, error) {
	return signToken(app, planKeyLabel, g)
}

// VerifyPlan accepts only a plan token Credimi signed for this workflow;
// every other token is ErrForbidden. Plans do not expire: activities accept
// them only while their session runs.
func VerifyPlan(app core.App, token, authorWorkflowID string) (PlanGrant, error) {
	var grant PlanGrant
	if err := verifyToken(app, planKeyLabel, token, &grant); err != nil {
		if errors.Is(err, errUnsigned) {
			return PlanGrant{}, fmt.Errorf("%w: plan not signed by Credimi", ErrForbidden)
		}
		return PlanGrant{}, err
	}
	if grant.AuthorWorkflowID != authorWorkflowID {
		return PlanGrant{}, fmt.Errorf("%w: plan for another session", ErrForbidden)
	}
	return grant, nil
}

// PlanSummary describes a normalized plan in lines a human confirms.
func PlanSummary(s Spec, prerequisites []ResolvedPrerequisite) []string {
	lines := []string{"Category: " + SlotCategory(s.Slot)}
	if s.Outcome == OutcomeRejection {
		lines = append(lines, "Outcome: the wallet rejects the flow")
	} else {
		lines = append(lines, "Outcome: success")
	}
	if s.Browser {
		lines = append(lines, "Runs in Chrome")
	}
	if len(prerequisites) == 0 {
		lines = append(lines, "Starts from: a freshly installed app")
	}
	for _, prerequisite := range prerequisites {
		lines = append(
			lines,
			"Starts from: "+prerequisite.Name+" ("+prerequisite.Action+")",
		)
	}
	for _, parameter := range s.Parameters {
		lines = append(
			lines,
			"${"+parameter.Name+"}: "+ParameterDescription(parameter.Source),
		)
	}
	return lines
}
