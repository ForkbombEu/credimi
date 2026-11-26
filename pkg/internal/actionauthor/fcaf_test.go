// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/walletroles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fcafTestSuite = "wallet_solution/relying_party"

// fcafTestScope is the relying-party scope of the repository contract.
func fcafTestScope(t *testing.T) *FCAFScope {
	t.Helper()
	const root = "../../../config_templates/fcaf"
	contract, err := walletroles.Load(root)
	require.NoError(t, err)
	return &FCAFScope{Contract: contract, Root: root, Suite: fcafTestSuite}
}

func deeplinkParameter() []ParameterSpec {
	return []ParameterSpec{{
		Name:   "deeplink",
		Source: SourceSpec{Kind: SourceLiteral, Value: "openid4vp://request"},
	}}
}

// haipPlan is a valid plan of fcaf-engagement-haip-vp.
func haipPlan() Spec {
	return Spec{
		Slot:    "verify-credential",
		Outcome: OutcomeSuccess,
		Prerequisites: []PrerequisiteSpec{
			{Action: "org/wallet/fcaf-reset-and-onboard"},
			{Candidate: "a1"},
		},
		Parameters: deeplinkParameter(),
		FCAFRole:   "fcaf-engagement-haip-vp",
	}
}

func TestFCAFOutcome(t *testing.T) {
	cases := map[walletroles.Outcome]string{
		walletroles.OutcomeSuccess:   OutcomeSuccess,
		walletroles.OutcomeRejection: OutcomeRejection,
		walletroles.OutcomeTolerant:  OutcomeTolerant,
		walletroles.OutcomeWait:      OutcomeWait,
	}
	for outcome, want := range cases {
		assert.Equal(t, want, FCAFOutcome(outcome), outcome)
	}
}

func TestCheckFCAFPlan(t *testing.T) {
	scope := fcafTestScope(t)
	cases := []struct {
		name    string
		chat    bool
		edit    func(*Spec)
		message string
	}{
		{name: "valid with a candidate", edit: func(*Spec) {}},
		{
			name: "valid with saved prerequisites",
			edit: func(s *Spec) {
				s.Prerequisites[1] = PrerequisiteSpec{Action: "org/wallet/fcaf-issue-pid"}
			},
		},
		{
			name: "valid onboarding",
			edit: func(s *Spec) {
				*s = Spec{
					Slot:     "onboarding",
					Outcome:  OutcomeSuccess,
					FCAFRole: "fcaf-reset-and-onboard",
				}
			},
		},
		{
			name: "valid wait",
			edit: func(s *Spec) {
				*s = Spec{
					Slot:          "other",
					Outcome:       OutcomeWait,
					WaitSeconds:   240,
					Prerequisites: []PrerequisiteSpec{{Candidate: "a1"}},
					FCAFRole:      "fcaf-wait-past-status-list-expiry",
				}
			},
		},
		{
			name: "valid replace",
			edit: func(s *Spec) {
				s.Replaces = &ReplaceSpec{Action: "org/wallet/fcaf-engagement-haip-vp"}
			},
		},
		{
			name:    "no FCAF session",
			chat:    true,
			edit:    func(*Spec) {},
			message: "fcaf_role: this session does not author FCAF actions",
		},
		{
			name:    "unknown role",
			edit:    func(s *Spec) { s.FCAFRole = "fcaf-unknown" },
			message: "fcaf_role: fcaf-unknown is not an FCAF role",
		},
		{
			name: "role of another suite",
			edit: func(s *Spec) {
				s.FCAFRole = "fcaf-engage-credential-offer"
			},
			message: "fcaf_role: fcaf-engage-credential-offer is not part of the " +
				"wallet_solution/relying_party suite",
		},
		{
			name: "wrong slot",
			edit: func(s *Spec) { s.Slot = "other" },
			message: "fcaf_role: fcaf-engagement-haip-vp is a verify-credential action: " +
				"use the verify-credential slot",
		},
		{
			name:    "wrong outcome",
			edit:    func(s *Spec) { s.Outcome = OutcomeRejection },
			message: "fcaf_role: fcaf-engagement-haip-vp has the success outcome",
		},
		{
			name: "wrong wait",
			edit: func(s *Spec) {
				*s = Spec{
					Slot:          "other",
					Outcome:       OutcomeWait,
					WaitSeconds:   20,
					Prerequisites: []PrerequisiteSpec{{Candidate: "a1"}},
					FCAFRole:      "fcaf-wait-past-status-list-expiry",
				}
			},
			message: "fcaf_role: fcaf-wait-past-status-list-expiry waits 240 s",
		},
		{
			name: "wrong parameters",
			edit: func(s *Spec) {
				s.Parameters = append(s.Parameters, ParameterSpec{
					Name:   "pin",
					Source: SourceSpec{Kind: SourceLiteral, Value: "123456"},
				})
			},
			message: "fcaf_role: fcaf-engagement-haip-vp takes the parameters [deeplink]",
		},
		{
			name: "missing prerequisite",
			edit: func(s *Spec) { s.Prerequisites = s.Prerequisites[:1] },
			message: "fcaf_role: fcaf-engagement-haip-vp starts from " +
				"[fcaf-reset-and-onboard fcaf-issue-pid], in that order",
		},
		{
			name: "prerequisites out of order",
			edit: func(s *Spec) {
				s.Prerequisites = []PrerequisiteSpec{
					{Action: "org/wallet/fcaf-issue-pid"},
					{Action: "org/wallet/fcaf-reset-and-onboard"},
				}
			},
			message: "fcaf_role: fcaf-engagement-haip-vp starts from " +
				"[fcaf-reset-and-onboard fcaf-issue-pid], in that order",
		},
		{
			name: "replaces another action",
			edit: func(s *Spec) {
				s.Replaces = &ReplaceSpec{Action: "org/wallet/verify-pid"}
			},
			message: "replaces: an FCAF fcaf-engagement-haip-vp action replaces only " +
				"the wallet's fcaf-engagement-haip-vp action",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := haipPlan()
			tc.edit(&spec)
			current := scope
			if tc.chat {
				current = nil
			}
			role, err := CheckFCAFPlan(current, &spec)
			if tc.message == "" {
				require.NoError(t, err)
				assert.Equal(t, spec.FCAFRole, role.Name)
				return
			}
			require.ErrorIs(t, err, ErrInvalid)
			assert.Equal(t, ErrInvalid.Error()+": "+tc.message, err.Error())
		})
	}
}
