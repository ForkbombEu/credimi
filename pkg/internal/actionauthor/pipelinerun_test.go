// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePipelineRunReference(t *testing.T) {
	tests := []struct {
		text string
		want PipelineRunRef
	}{
		{
			text: "https://credimi.io/my/tests/runs/Pipeline-demo-1/run-1",
			want: PipelineRunRef{WorkflowID: "Pipeline-demo-1", RunID: "run-1"},
		},
		{
			text: "https://credimi.io/it/my/tests/runs/Pipeline-demo-1/?tab=logs",
			want: PipelineRunRef{WorkflowID: "Pipeline-demo-1"},
		},
		{
			text: "http://localhost:5100/my/pipelines/acme/onboarding-check",
			want: PipelineRunRef{Pipeline: "acme/onboarding-check"},
		},
		{
			text: "https://credimi.io/hub/pipelines/acme/onboarding-check/",
			want: PipelineRunRef{Pipeline: "acme/onboarding-check"},
		},
		{
			text: " Pipeline-Sched-demo-2 ",
			want: PipelineRunRef{WorkflowID: "Pipeline-Sched-demo-2"},
		},
		{text: "acme/onboarding-check", want: PipelineRunRef{Pipeline: "acme/onboarding-check"}},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got, err := ParsePipelineRunReference(tt.text)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	for _, text := range []string{
		"",
		"fix my pipeline",
		"Acme/Onboarding Check",
		"acme/wallet/action",
		"https://credimi.io/my/tests/runs/",
		"https://credimi.io/my/tests/runs/Pipeline-demo-1/run-1/extra",
		"https://credimi.io/my/pipelines/acme",
		"https://credimi.io/my/wallets/acme/wallet",
		"Pipeline-demo/run",
	} {
		t.Run("invalid "+text, func(t *testing.T) {
			_, err := ParsePipelineRunReference(text)
			require.ErrorIs(t, err, ErrInvalid)
			require.ErrorContains(
				t,
				err,
				"not a pipeline run or pipeline: "+strings.TrimSpace(text),
			)
		})
	}
}

func TestMaestroFailure(t *testing.T) {
	t.Run("failed command", func(t *testing.T) {
		stdout := strings.Join([]string{
			"WARNING: A restricted method in java.lang.System has been called",
			"SLF4J: No SLF4J providers were found.",
			"kotlin-logging: initializing",
			"Running on emulator-5554",
			"",
			"Launch app \"eu.wallet\"... COMPLETED",
			"Assert that \"MY WALLET\" is visible... FAILED",
			"Tap on \"Continue\"... FAILED",
			"   ",
			"Element not found: Text matching regex: MY WALLET\r",
		}, "\n")
		command, tail := MaestroFailure(stdout)
		assert.Equal(t, `Assert that "MY WALLET" is visible`, command)
		assert.Equal(t, strings.Join([]string{
			"Running on emulator-5554",
			"Launch app \"eu.wallet\"... COMPLETED",
			"Assert that \"MY WALLET\" is visible... FAILED",
			"Tap on \"Continue\"... FAILED",
			"Element not found: Text matching regex: MY WALLET",
		}, "\n"), tail)
	})

	t.Run("no failed command", func(t *testing.T) {
		command, tail := MaestroFailure("Launch app... COMPLETED\nTap on \"OK\"... WARNED\n")
		assert.Empty(t, command)
		assert.Equal(t, "Launch app... COMPLETED\nTap on \"OK\"... WARNED", tail)
	})

	t.Run("noise only and long output", func(t *testing.T) {
		command, tail := MaestroFailure("WARNING: x\nSLF4J: y\n")
		assert.Empty(t, command)
		assert.Empty(t, tail)

		lines := make([]string, 0, 60)
		for i := range 60 {
			lines = append(lines, fmt.Sprintf("line %d", i))
		}
		_, tail = MaestroFailure(strings.Join(lines, "\n"))
		assert.Equal(t, strings.Join(lines[20:], "\n"), tail)

		long := strings.Repeat(strings.Repeat("é", 199)+"\n", 30) + "end"
		_, tail = MaestroFailure(long)
		assert.Len(t, []rune(tail), 4000)
		assert.True(t, strings.HasSuffix(long, tail))
	})
}

func TestSlotForCategory(t *testing.T) {
	for category, want := range map[string]string{
		"get-credential-generic":  "get-credential",
		"get-credential-specific": "get-credential-specific",
		"onboarding":              "onboarding",
		SlotInstallApp:            SlotInstallApp,
		"verify-credential":       "verify-credential",
		"other":                   "other",
	} {
		slot, ok := SlotForCategory(category)
		assert.True(t, ok, category)
		assert.Equal(t, want, slot, category)
	}
	_, ok := SlotForCategory("legacy")
	assert.False(t, ok)
}

// mobileStep is a mobile-automation step running action with parameters.
func mobileStep(id, action string, parameters map[string]any) pipeline.StepDefinition {
	payload := map[string]any{"action_id": action, "version_id": "acme/wallet/v1"}
	if parameters != nil {
		payload["parameters"] = parameters
	}
	return pipelineStep(id, StepUseMobileAutomation, payload)
}

func pipelineStep(id, use string, payload map[string]any) pipeline.StepDefinition {
	return pipeline.StepDefinition{StepSpec: pipeline.StepSpec{
		ID:   id,
		Use:  use,
		With: pipeline.StepInputs{Payload: payload},
	}}
}

// storeSelection selects a new store wallet of the selection's organization.
func storeSelection(
	t *testing.T,
	app *tests.TestApp,
	selection Selection,
	namespace string,
) Selection {
	t.Helper()
	identifier := seedStoreWallet(t, app, selection, "Play wallet", map[string]any{
		"google_app_id": "com.acme.wallet",
	})
	store, err := Select(app, namespace, BuildRequest{
		Actor:                   selection.Actor,
		WalletVersionIdentifier: identifier,
		Platform:                PlatformAndroid,
	})
	require.NoError(t, err)
	require.Equal(t, SourceStore, store.Source)
	return store
}

const testActionCode = "appId: eu.wallet\n---\n- launchApp\n"

func TestSuggestPlanFromAStorePipeline(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	installer, err := Select(app, namespace, request)
	require.NoError(t, err)
	store := storeSelection(t, app, installer, namespace)
	org := store.Organization.Id
	_, install := seedWalletAction(
		t,
		app,
		org,
		store.Wallet,
		"Install",
		SlotInstallApp,
		testActionCode,
		false,
	)
	_, onboard := seedWalletAction(
		t,
		app,
		org,
		store.Wallet,
		"Onboard",
		"onboarding",
		testActionCode,
		false,
	)
	_, otherWallet := seedWalletAction(
		t,
		app,
		org,
		installer.Wallet,
		"Other onboard",
		"onboarding",
		testActionCode,
		false,
	)
	get, getPath := seedWalletAction(
		t,
		app,
		org,
		store.Wallet,
		"Get PID",
		"get-credential-generic",
		testActionCode,
		false,
	)
	definition := &pipeline.WorkflowDefinition{Steps: []pipeline.StepDefinition{
		mobileStep("install", install, nil),
		mobileStep("onboard", onboard, map[string]any{"PIN": "123456"}),
		mobileStep("elsewhere", otherWallet, nil),
		pipelineStep(
			"offer",
			"credential-offer",
			map[string]any{"credential_id": "acme/issuer/pid"},
		),
		pipelineStep("verify", "use-case-verification-deeplink", map[string]any{
			"use_case_id": "acme/verifier/age",
		}),
		pipelineStep("session", "http-request", map[string]any{
			"method":  "POST",
			"url":     "https://issuer.example/sessions",
			"headers": map[string]any{"Content-Type": "application/json"},
			"body":    map[string]any{"type": "pid"},
		}),
		pipelineStep(
			"inline",
			StepUseMobileAutomation,
			map[string]any{"action_code": testActionCode},
		),
		mobileStep("get", getPath, map[string]any{
			"OFFER":        "${{ offer.outputs }}",
			"REQUEST":      "${{verify.outputs.deeplink}}",
			"LINK":         "${{ session.outputs.body.data.deeplink }}",
			"NAME":         "Mario",
			"MIXED":        "id-${{ onboard.outputs.id }}",
			"FROM_INSTALL": "${{ install.outputs }}",
		}),
	}}

	plan, notes, err := SuggestPlan(app, definition, 7, get, store)
	require.NoError(t, err)
	assert.Equal(t, Spec{
		Slot:     "get-credential",
		Outcome:  OutcomeSuccess,
		Replaces: &ReplaceSpec{Action: getPath},
		Prerequisites: []PrerequisiteSpec{{
			Action: onboard,
			Parameters: []ParameterSpec{{
				Name:   "PIN",
				Source: SourceSpec{Kind: SourceLiteral, Value: "123456"},
			}},
		}},
		Parameters: []ParameterSpec{
			{Name: "LINK", Source: SourceSpec{
				Kind: SourceHTTP,
				Request: &HTTPSourceRequest{
					Method:  "POST",
					URL:     "https://issuer.example/sessions",
					Headers: map[string]string{"Content-Type": "application/json"},
					Body:    map[string]any{"type": "pid"},
				},
				ValuePath: "data.deeplink",
			}},
			{Name: "NAME", Source: SourceSpec{Kind: SourceLiteral, Value: "Mario"}},
			{Name: "OFFER", Source: SourceSpec{
				Kind:       SourceCredentialOffer,
				Identifier: "acme/issuer/pid",
			}},
			{Name: "REQUEST", Source: SourceSpec{
				Kind:       SourceUseCaseVerification,
				Identifier: "acme/verifier/age",
			}},
		},
	}, plan)
	assert.Equal(t, []string{
		"parameter FROM_INSTALL comes from step install (mobile-automation): describe its source",
		"parameter MIXED comes from step onboard (mobile-automation): describe its source",
	}, notes)

	definition.Steps[7].With.Payload["version_id"] = ExternalSourceVersion
	described, err := DescribeFailedStep(app, store, definition, 7, StepFailure{
		Code:    "CRE301",
		Message: "Maestro flow failed.",
		Stdout:  "Tap on \"Add\"... FAILED\n",
	})
	require.NoError(t, err)
	assert.Equal(t, PipelineRunFailure{
		StepID:          "get",
		Action:          getPath,
		ActionName:      "Get PID",
		Candidates:      []string{},
		Owned:           true,
		InSessionWallet: true,
		Version:         ExternalSourceVersion,
		Code:            "CRE301",
		Message:         "Maestro flow failed.",
		FailedCommand:   `Tap on "Add"`,
		OutputTail:      `Tap on "Add"... FAILED`,
		Plan:            &plan,
		Notes:           notes,
	}, described)
}

func TestSuggestPlanForAForeignAction(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	selection, err := Select(app, namespace, request)
	require.NoError(t, err)
	other, err := app.FindFirstRecordByData("organizations", "name", "userB's organization")
	require.NoError(t, err)
	wallets, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	wallet := core.NewRecord(wallets)
	wallet.Set("owner", other.Id)
	wallet.Set("name", "Published wallet")
	require.NoError(t, app.Save(wallet))
	_, onboard := seedWalletAction(
		t,
		app,
		other.Id,
		wallet,
		"Onboard",
		"onboarding",
		testActionCode,
		true,
	)
	verify, verifyPath := seedWalletAction(
		t,
		app,
		other.Id,
		wallet,
		"Verify",
		"verify-credential",
		testActionCode,
		true,
	)
	definition := &pipeline.WorkflowDefinition{Steps: []pipeline.StepDefinition{
		mobileStep("onboard", onboard, nil),
		mobileStep("verify", verifyPath, nil),
	}}

	plan, notes, err := SuggestPlan(app, definition, 1, verify, selection)
	require.NoError(t, err)
	assert.Equal(t, Spec{
		Slot:          "verify-credential",
		Outcome:       OutcomeSuccess,
		Prerequisites: []PrerequisiteSpec{{Action: onboard, Parameters: []ParameterSpec{}}},
		Parameters:    []ParameterSpec{},
	}, plan)
	assert.Empty(t, notes)

	described, err := DescribeFailedStep(app, selection, definition, 1, StepFailure{Code: "CRE301"})
	require.NoError(t, err)
	assert.False(t, described.Owned)
	assert.False(t, described.InSessionWallet)
	assert.Equal(t, "Verify", described.ActionName)
	assert.Equal(t, []string{
		notOwnedNote,
		"this action is not of the session's wallet: start a session on " +
			other.GetString("canonified_name") + "/" + wallet.GetString("canonified_name"),
		"the run used version acme/wallet/v1, the session uses " + selection.Identifier,
	}, described.Notes)
	assert.Equal(t, &plan, described.Plan)
}

func TestSuggestPlanForAnOnboardingAction(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	selection, err := Select(app, namespace, request)
	require.NoError(t, err)
	org := selection.Organization.Id
	_, first := seedWalletAction(
		t,
		app,
		org,
		selection.Wallet,
		"First",
		"onboarding",
		testActionCode,
		false,
	)
	onboard, onboardPath := seedWalletAction(
		t,
		app,
		org,
		selection.Wallet,
		"Onboard",
		"onboarding",
		testActionCode,
		false,
	)
	definition := &pipeline.WorkflowDefinition{Steps: []pipeline.StepDefinition{
		mobileStep("first", first, nil),
		mobileStep("onboard", onboardPath, map[string]any{"PIN": "${{ first.outputs }}"}),
	}}

	plan, notes, err := SuggestPlan(app, definition, 1, onboard, selection)
	require.NoError(t, err)
	assert.Equal(t, Spec{
		Slot:       "onboarding",
		Outcome:    OutcomeSuccess,
		Replaces:   &ReplaceSpec{Action: onboardPath},
		Parameters: []ParameterSpec{},
	}, plan)
	assert.Equal(t, []string{
		"parameter PIN comes from step first (mobile-automation): describe its source",
	}, notes)

	legacy := core.NewRecord(onboard.Collection())
	legacy.Set("owner", org)
	legacy.Set("wallet", selection.Wallet.Id)
	legacy.Set("category", "legacy")
	plan, notes, err = SuggestPlan(app, definition, 0, legacy, selection)
	require.NoError(t, err)
	assert.Equal(t, "other", plan.Slot)
	assert.Equal(t, []string{"action category legacy has no slot: the plan uses other"}, notes)
}

func TestDescribeFailedStepWithoutOneStoredAction(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	selection, err := Select(app, namespace, request)
	require.NoError(t, err)
	definition := &pipeline.WorkflowDefinition{Steps: []pipeline.StepDefinition{
		mobileStep("gone", "acme/wallet/gone", nil),
		pipelineStep("many", StepUseMobileAutomation, map[string]any{"actions": []any{
			map[string]any{"id": "acme/wallet/first"},
			map[string]any{"code": testActionCode},
			map[string]any{"id": "/acme/wallet/second"},
		}}),
		pipelineStep(
			"inline",
			StepUseMobileAutomation,
			map[string]any{"action_code": testActionCode},
		),
	}}

	gone, err := DescribeFailedStep(app, selection, definition, 0, StepFailure{Code: "CRE301"})
	require.NoError(t, err)
	assert.Equal(t, "acme/wallet/gone", gone.Action)
	assert.Nil(t, gone.Plan)
	assert.Equal(t, []string{"action acme/wallet/gone no longer exists"}, gone.Notes)

	many, err := DescribeFailedStep(app, selection, definition, 1, StepFailure{Code: "CRE301"})
	require.NoError(t, err)
	assert.Empty(t, many.Action)
	assert.Equal(t, []string{"acme/wallet/first", "acme/wallet/second"}, many.Candidates)
	assert.Nil(t, many.Plan)

	inline, err := DescribeFailedStep(app, selection, definition, 2, StepFailure{Code: "CRE301"})
	require.NoError(t, err)
	assert.Empty(t, inline.Action)
	assert.Empty(t, inline.Candidates)
	assert.Nil(t, inline.Plan)
	assert.Empty(t, inline.Notes)
}
