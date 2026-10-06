// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const covRunnerCIUseCaseStepYAML = "name: test\nsteps:\n" +
	"  - id: verify\n    use: use-case-verification-deeplink\n    with:\n      use_case_id: %s\n" +
	"    on_error:\n      - id: verify-err\n        use: use-case-verification-deeplink\n        with:\n          use_case_id: %s\n" +
	"    on_success:\n      - id: verify-ok\n        use: use-case-verification-deeplink\n        with:\n          use_case_id: %s\n" +
	"  - id: other\n    use: http-request\n    with:\n      use_case_id: %s\n"

func covRunnerCIUseCasePipeline(main, onErr, onSuccess, other string) string {
	return fmt.Sprintf(covRunnerCIUseCaseStepYAML, main, onErr, onSuccess, other)
}

func covRunnerStubHealth(t *testing.T, fn func(context.Context, string) (bool, error)) {
	t.Helper()
	orig := checkRunnerReachable
	t.Cleanup(func() { checkRunnerReachable = orig })
	checkRunnerReachable = fn
}

func covRunnerStubQueue(
	t *testing.T,
	fn func(context.Context, string) (workflows.MobileDeviceSemaphoreStateView, error),
) {
	t.Helper()
	orig := queryMobileDeviceSemaphoreState
	t.Cleanup(func() { queryMobileDeviceSemaphoreState = orig })
	queryMobileDeviceSemaphoreState = fn
}

func covRunnerCIEvent(t *testing.T, contentType, body string) *core.RequestEvent {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/pipeline/run", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return &core.RequestEvent{Event: router.Event{Request: req, Response: httptest.NewRecorder()}}
}

func TestParsePipelineCIBaseRequest(t *testing.T) {
	testCases := []struct {
		name        string
		contentType string
		body        string
		want        pipelineCIBaseRequest
		wantReason  string
	}{
		{
			name:        "json with ids array and explicit commit",
			contentType: "application/json; charset=utf-8",
			body: `{"pipeline_identifier":" org/p ","commit_sha":" abc ",` +
				`"use_case_ids":["/org/a", "org/b,org/a", ""],"device_id":" dev ",` +
				`"device_type":" redroid ","verifier_host_url":" https://h.example "}`,
			want: pipelineCIBaseRequest{
				PipelineIdentifier: "org/p",
				CommitSHA:          "abc",
				Metadata:           map[string]any{"sha": "abc"},
				IDs:                []string{"org/a", "org/b"},
				DeviceID:           "dev",
				DeviceType:         "redroid",
				HostURL:            "https://h.example",
			},
		},
		{
			name:        "json with single string id and sha from metadata",
			contentType: "application/json",
			body:        `{"pipeline_identifier":"org/p","metadata":{"sha":"fromMeta","pr":1},"use_case_ids":"org/x"}`,
			want: pipelineCIBaseRequest{
				PipelineIdentifier: "org/p",
				CommitSHA:          "fromMeta",
				Metadata:           map[string]any{"sha": "fromMeta", "pr": float64(1)},
				IDs:                []string{"org/x"},
			},
		},
		{
			name:        "json explicit commit does not override metadata sha",
			contentType: "application/json",
			body:        `{"pipeline_identifier":"org/p","commit_sha":"c1","metadata":{"sha":"m1"},"use_case_ids":5}`,
			want: pipelineCIBaseRequest{
				PipelineIdentifier: "org/p",
				CommitSHA:          "c1",
				Metadata:           map[string]any{"sha": "m1"},
			},
		},
		{
			name:        "invalid json body",
			contentType: "application/json",
			body:        `{`,
			wantReason:  "invalid JSON input",
		},
		{
			name:        "json metadata that is not an object",
			contentType: "application/json",
			body:        `{"metadata":"nope"}`,
			wantReason:  "metadata must be valid JSON",
		},
		{
			name:        "form with repeated and comma separated ids",
			contentType: "application/x-www-form-urlencoded",
			body: "pipeline_identifier=org%2Fp&use_case_ids=org%2Fa%2Corg%2Fb&use_case_ids=org%2Fa" +
				"&metadata=%7B%22sha%22%3A%22fsha%22%7D&device_type=redroid&verifier_host_url=https%3A%2F%2Fh",
			want: pipelineCIBaseRequest{
				PipelineIdentifier: "org/p",
				CommitSHA:          "fsha",
				Metadata:           map[string]any{"sha": "fsha"},
				IDs:                []string{"org/a", "org/b"},
				DeviceType:         "redroid",
				HostURL:            "https://h",
			},
		},
		{
			name:        "form without metadata gets sha from commit_sha",
			contentType: "application/x-www-form-urlencoded",
			body:        "pipeline_identifier=org%2Fp&commit_sha=c2",
			want: pipelineCIBaseRequest{
				PipelineIdentifier: "org/p",
				CommitSHA:          "c2",
				Metadata:           map[string]any{"sha": "c2"},
			},
		},
		{
			name:        "form with invalid metadata",
			contentType: "application/x-www-form-urlencoded",
			body:        "metadata=%7Bbad",
			wantReason:  "metadata must be valid JSON",
		},
		{
			name:        "malformed form encoding",
			contentType: "application/x-www-form-urlencoded",
			body:        "pipeline_identifier=%zz",
			wantReason:  "failed to parse form request",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, apiErr := parsePipelineCIBaseRequest(
				covRunnerCIEvent(t, tc.contentType, tc.body),
				"use_case_ids",
				"verifier_host_url",
			)
			if tc.wantReason != "" {
				require.NotNil(t, apiErr)
				assert.Equal(t, http.StatusBadRequest, apiErr.Code)
				assert.Equal(t, tc.wantReason, apiErr.Reason)
				return
			}
			require.Nil(t, apiErr)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestValidatePipelineCIBaseRequest(t *testing.T) {
	valid := pipelineCIBaseRequest{
		PipelineIdentifier: "org/p",
		CommitSHA:          "abc",
		HostURL:            "https://host.example",
	}
	testCases := []struct {
		name       string
		mutate     func(*pipelineCIBaseRequest)
		wantDomain string
		wantReason string
	}{
		{name: "valid request", mutate: func(*pipelineCIBaseRequest) {}},
		{
			name:       "valid with allowed device type",
			mutate:     func(r *pipelineCIBaseRequest) { r.DeviceType = "ios_simulator"; r.HostURL = "http://h" },
			wantReason: "",
		},
		{
			name:       "missing pipeline identifier",
			mutate:     func(r *pipelineCIBaseRequest) { r.PipelineIdentifier = "  " },
			wantDomain: "pipeline_identifier",
			wantReason: "pipeline_identifier is required",
		},
		{
			name:       "missing commit sha",
			mutate:     func(r *pipelineCIBaseRequest) { r.CommitSHA = "" },
			wantDomain: "metadata",
			wantReason: "commit_sha or metadata.sha is required",
		},
		{
			name:       "host url without scheme",
			mutate:     func(r *pipelineCIBaseRequest) { r.HostURL = "host.example" },
			wantDomain: "host_url",
			wantReason: "host_url is invalid",
		},
		{
			name:       "host url with unsupported scheme",
			mutate:     func(r *pipelineCIBaseRequest) { r.HostURL = "ftp://host.example" },
			wantDomain: "host_url",
			wantReason: "host_url is invalid",
		},
		{
			name:       "unknown device type",
			mutate:     func(r *pipelineCIBaseRequest) { r.DeviceType = "toaster" },
			wantDomain: "device_type",
			wantReason: "device_type is invalid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := valid
			tc.mutate(&input)
			apiErr := validatePipelineCIBaseRequest(input, "host_url")
			if tc.wantReason == "" {
				assert.Nil(t, apiErr)
				return
			}
			require.NotNil(t, apiErr)
			assert.Equal(t, http.StatusBadRequest, apiErr.Code)
			assert.Equal(t, tc.wantDomain, apiErr.Domain)
			assert.Equal(t, tc.wantReason, apiErr.Reason)
		})
	}
}

func TestResolvePipelineCIRunContext(t *testing.T) {
	app := setupPipelineVerifierCIApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	otherOrgID, err := getOrgIDfromName("userB's organization")
	require.NoError(t, err)

	createWalletAPITestPipelineNamed(t, app, orgID, "owned", "name: owned\nsteps: []\n", false)
	blank := createWalletAPITestPipelineNamed(t, app, orgID, "blank", "name: b\nsteps: []\n", false)
	blankWalletAPITestPipelineYAML(t, app, blank.Id)
	createWalletAPITestPipelineNamed(t, app, otherOrgID, "foreign", "name: f\nsteps: []\n", false)
	createWalletAPITestPipelineNamed(t, app, otherOrgID, "shared", "name: s\nsteps: []\n", true)

	testCases := []struct {
		name       string
		auth       *core.Record
		identifier string
		wantStatus int
		wantYAML   string
	}{
		{
			name:       "anonymous",
			identifier: "usera-s-organization/owned",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "unknown pipeline",
			auth:       user,
			identifier: "usera-s-organization/ghost",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "foreign private pipeline",
			auth:       user,
			identifier: "userb-s-organization/foreign",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "pipeline without yaml",
			auth:       user,
			identifier: "usera-s-organization/blank",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "owned pipeline",
			auth:       user,
			identifier: "usera-s-organization/owned",
			wantYAML:   "name: owned\nsteps: []",
		},
		{
			name:       "foreign published pipeline",
			auth:       user,
			identifier: "userb-s-organization/shared",
			wantYAML:   "name: s\nsteps: []",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := &core.RequestEvent{App: app, Auth: tc.auth}
			runContext, apiErr := resolvePipelineCIRunContext(event, tc.identifier)
			if tc.wantStatus != 0 {
				require.NotNil(t, apiErr)
				assert.Equal(t, tc.wantStatus, apiErr.Code)
				return
			}
			require.Nil(t, apiErr)
			assert.Equal(t, orgID, runContext.OrganizationRecord.Id)
			assert.Equal(t, user.Id, runContext.UserID)
			assert.Equal(t, "userA@example.org", runContext.UserEmail)
			assert.Equal(t, tc.wantYAML, runContext.PipelineYAML)
		})
	}
}

func TestCreatePipelineCITempRecordsRewritesHostsAndRollsBack(t *testing.T) {
	app := setupPipelineVerifierCIApp(t)
	defer app.Cleanup()
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	first := createVerifierCIUseCase(t, app, orgID, "Verifier One", "Case One")
	second := createVerifierCIUseCase(t, app, orgID, "Verifier Two", "Case Two")
	event := &core.RequestEvent{App: app}
	opts := pipelineCITempRecordsOptions{
		Collection:     "use_cases_verifications",
		IdentifierKey:  "use_case_id",
		IdentifierName: "use case verification",
		ResourceDomain: "use_case_verification",
		ResourceName:   "use case verification",
		OwnerID:        orgID,
		CommitSHA:      "ABC123",
		HostURL:        "https://ci.example/run",
	}

	countUseCases := func() int {
		records, err := app.FindAllRecords("use_cases_verifications")
		require.NoError(t, err)
		return len(records)
	}
	before := countUseCases()

	t.Run("creates one temp record per ref with rewritten host", func(t *testing.T) {
		opts := opts
		opts.Refs = []string{first, "/" + second}
		temps, rewriteMap, apiErr := createPipelineCITempRecords(event, opts)
		require.Nil(t, apiErr)
		require.Len(t, temps, 2)
		assert.Len(t, rewriteMap, 2)
		assert.Equal(t, temps[0].Identifier, rewriteMap[first])
		assert.Equal(t, temps[1].Identifier, rewriteMap[second])
		for _, temp := range temps {
			assert.Equal(t, orgID, temp.Record.GetString("owner"))
			assert.False(t, temp.Record.GetBool("published"))
			assert.Contains(t, temp.Record.GetString("name"), "-abc123")
			assert.Contains(t, temp.Record.GetString("yaml"), "host: https://ci.example/run")
			assert.NotContains(t, temp.Record.GetString("yaml"), "verifier.example/old")
		}
		rollbackPipelineCITempRecords(event, temps, "use case verification")
		assert.Equal(t, before, countUseCases())
	})

	t.Run("a later invalid ref rolls back earlier temp records", func(t *testing.T) {
		opts := opts
		opts.Refs = []string{first, "usera-s-organization/missing/case"}
		temps, rewriteMap, apiErr := createPipelineCITempRecords(event, opts)
		require.NotNil(t, apiErr)
		assert.Equal(t, http.StatusBadRequest, apiErr.Code)
		assert.Nil(t, temps)
		assert.Nil(t, rewriteMap)
		assert.Equal(t, before, countUseCases())
	})

	t.Run("a foreign private ref is rejected after rollback", func(t *testing.T) {
		otherOrgID, err := getOrgIDfromName("userB's organization")
		require.NoError(t, err)
		opts := opts
		opts.OwnerID = otherOrgID
		opts.Refs = []string{first}
		_, _, apiErr := createPipelineCITempRecords(event, opts)
		require.NotNil(t, apiErr)
		assert.Equal(t, http.StatusForbidden, apiErr.Code)
		assert.Equal(t, before, countUseCases())
	})

	t.Run("save failure on second ref rolls back the first", func(t *testing.T) {
		var mu sync.Mutex
		creates := 0
		hookID := app.OnRecordCreate("use_cases_verifications").
			BindFunc(func(e *core.RecordEvent) error {
				mu.Lock()
				creates++
				n := creates
				mu.Unlock()
				if n > 1 {
					return errors.New("boom")
				}
				return e.Next()
			})
		defer app.OnRecordCreate("use_cases_verifications").Unbind(hookID)

		opts := opts
		opts.Refs = []string{first, second}
		_, _, apiErr := createPipelineCITempRecords(event, opts)
		require.NotNil(t, apiErr)
		assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
		assert.Equal(t, "failed to create temporary use case verification", apiErr.Reason)
		assert.Equal(t, before, countUseCases())
	})
}

func TestRewritePipelineCIStepRefsYAML(t *testing.T) {
	t.Run("empty rewrite map returns input unchanged", func(t *testing.T) {
		got, apiErr := rewritePipelineCIStepRefsYAML("not: [valid", nil, "x", "y")
		require.Nil(t, apiErr)
		assert.Equal(t, "not: [valid", got)
	})

	t.Run("invalid yaml with rewrites is a bad request", func(t *testing.T) {
		_, apiErr := rewritePipelineCIStepRefsYAML(
			"steps: [", map[string]string{"a": "b"}, "x", "y",
		)
		require.NotNil(t, apiErr)
		assert.Equal(t, http.StatusBadRequest, apiErr.Code)
	})

	t.Run(
		"rewrites main, on_error and on_success steps of the matching use only",
		func(t *testing.T) {
			pipelineYAML := covRunnerCIUseCasePipeline("org/v/a", "/org/v/a", "org/v/c", "org/v/a")
			got, apiErr := rewritePipelineCIStepRefsYAML(
				pipelineYAML,
				map[string]string{"org/v/a": "org/v/a-tmp"},
				"use-case-verification-deeplink",
				"use_case_id",
			)
			require.Nil(t, apiErr)
			def, apiErr := parsePipelineCIWorkflow(got)
			require.Nil(t, apiErr)
			require.Len(t, def.Steps, 2)
			assert.Equal(t, "org/v/a-tmp", def.Steps[0].With.Payload["use_case_id"])
			assert.Equal(t, "org/v/a-tmp", def.Steps[0].OnError[0].With.Payload["use_case_id"])
			assert.Equal(t, "org/v/c", def.Steps[0].OnSuccess[0].With.Payload["use_case_id"])
			assert.Equal(t, "org/v/a", def.Steps[1].With.Payload["use_case_id"])
		},
	)
}

func TestCollectPipelineCIReferences(t *testing.T) {
	assert.Nil(t, collectPipelineCIReferences(nil, "x", "y"))

	def, apiErr := parsePipelineCIWorkflow(
		covRunnerCIUseCasePipeline("/org/v/a", "org/v/b", "org/v/c", "org/v/d"),
	)
	require.Nil(t, apiErr)
	refs := collectPipelineCIReferences(def, "use-case-verification-deeplink", "use_case_id")
	assert.Equal(t, []string{"org/v/a", "org/v/b", "org/v/c"}, refs)
}

func TestPipelineCIMobileRunnerSelectionStateNestedSteps(t *testing.T) {
	testCases := []struct {
		name       string
		yaml       string
		wantStep   bool
		wantGlobal bool
	}{
		{
			name: "on_error mobile step without device needs global runner",
			yaml: "name: t\nsteps:\n  - id: a\n    use: http-request\n    with:\n      url: x\n" +
				"    on_error:\n      - id: b\n        use: mobile-automation\n        with:\n          action_id: x\n",
			wantGlobal: true,
		},
		{
			name: "on_success mobile step with device is a step runner",
			yaml: "name: t\nsteps:\n  - id: a\n    use: http-request\n    with:\n      url: x\n" +
				"    on_success:\n      - id: b\n        use: mobile-automation\n        with:\n          device_id: o/r/d\n",
			wantStep: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			def, apiErr := parsePipelineCIWorkflow(tc.yaml)
			require.Nil(t, apiErr)
			hasStep, needsGlobal := pipelineCIMobileRunnerSelectionState(def)
			assert.Equal(t, tc.wantStep, hasStep)
			assert.Equal(t, tc.wantGlobal, needsGlobal)
		})
	}
	hasStep, needsGlobal := pipelineCIMobileRunnerSelectionState(nil)
	assert.False(t, hasStep)
	assert.False(t, needsGlobal)
}

func TestNormalizePipelineCIIdentifiersAndFirstNonEmpty(t *testing.T) {
	assert.Equal(t,
		[]string{"a/b", "c"},
		normalizePipelineCIIdentifiers([]string{" /a/b , ,c", "a/b", ""}),
	)
	assert.Nil(t, normalizePipelineCIIdentifiers(nil))
	assert.Equal(t, "", firstNonEmpty(" ", ""))
	assert.Equal(t, "x", firstNonEmpty("", "x", "y"))
}

func TestResolvePipelineCIDeviceID(t *testing.T) {
	const mobileNoDevice = "name: t\nsteps:\n  - id: m\n    use: mobile-automation\n    with:\n      action_id: x\n"
	const noMobile = "name: t\nsteps:\n  - id: h\n    use: http-request\n    with:\n      url: x\n"

	app := setupPipelineWalletAPKApp(t)
	defer app.Cleanup()
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	createWalletAPKMobileRunner(t, app, orgID, "phone-runner", "android_phone", false)

	covRunnerStubQueue(
		t,
		func(context.Context, string) (workflows.MobileDeviceSemaphoreStateView, error) {
			return workflows.MobileDeviceSemaphoreStateView{}, errSemaphoreNotFound
		},
	)

	testCases := []struct {
		name       string
		yaml       string
		input      pipelineCIBaseRequest
		online     bool
		wantID     string
		wantGlobal bool
		wantStatus int
	}{
		{
			name:  "pipeline without mobile steps ignores device hints",
			yaml:  noMobile,
			input: pipelineCIBaseRequest{DeviceID: "x", DeviceType: "redroid"},
		},
		{
			name: "explicit device on online runner is used verbatim",
			yaml: mobileNoDevice,
			input: pipelineCIBaseRequest{
				DeviceID: " /usera-s-organization/phone-runner/device-1",
			},
			online:     true,
			wantID:     " /usera-s-organization/phone-runner/device-1",
			wantGlobal: true,
		},
		{
			name: "explicit device on offline runner",
			yaml: mobileNoDevice,
			input: pipelineCIBaseRequest{
				DeviceID: "usera-s-organization/phone-runner/device-1",
			},
			wantGlobal: true,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "explicit unknown device",
			yaml:       mobileNoDevice,
			input:      pipelineCIBaseRequest{DeviceID: "usera-s-organization/phone-runner/ghost"},
			online:     true,
			wantGlobal: true,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "global runner needed without device or type",
			yaml:       mobileNoDevice,
			wantGlobal: true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "device type selects an online device",
			yaml:       mobileNoDevice,
			input:      pipelineCIBaseRequest{DeviceType: "android_phone"},
			online:     true,
			wantID:     "usera-s-organization/phone-runner/device-1",
			wantGlobal: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			covRunnerStubHealth(
				t,
				func(context.Context, string) (bool, error) { return tc.online, nil },
			)
			def, apiErr := parsePipelineCIWorkflow(tc.yaml)
			require.Nil(t, apiErr)
			deviceID, hasStep, needsGlobal, apiErr := resolvePipelineCIDeviceID(
				context.Background(), app, orgID, def, tc.input,
			)
			assert.False(t, hasStep)
			assert.Equal(t, tc.wantGlobal, needsGlobal)
			if tc.wantStatus != 0 {
				require.NotNil(t, apiErr)
				assert.Equal(t, tc.wantStatus, apiErr.Code)
				assert.Empty(t, deviceID)
				return
			}
			require.Nil(t, apiErr)
			assert.Equal(t, tc.wantID, deviceID)
		})
	}
}

func TestSelectPipelineCIDeviceByTypePrefersShortestBacklog(t *testing.T) {
	app := setupPipelineVerifierCIApp(t)
	defer app.Cleanup()
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	createWalletAPKMobileRunner(t, app, orgID, "busy", "redroid", false)
	createWalletAPKMobileRunner(t, app, orgID, "idle", "redroid", false)
	createWalletAPKMobileRunner(t, app, orgID, "also-idle", "redroid", false)
	createWalletAPKMobileRunner(t, app, orgID, "down", "redroid", false)
	createWalletAPKMobileRunner(t, app, orgID, "emu", "android_emulator", false)

	covRunnerStubHealth(t, func(_ context.Context, runnerURL string) (bool, error) {
		return !strings.Contains(runnerURL, "down"), nil
	})

	backlogs := map[string]workflows.MobileDeviceSemaphoreStateView{
		"usera-s-organization/busy/device-1":      {QueueLen: 3},
		"usera-s-organization/idle/device-1":      {SlotsUsed: 1},
		"usera-s-organization/also-idle/device-1": {QueueLen: 1},
	}
	covRunnerStubQueue(
		t,
		func(_ context.Context, id string) (workflows.MobileDeviceSemaphoreStateView, error) {
			return backlogs[id], nil
		},
	)

	deviceID, apiErr := selectPipelineCIDeviceByType(context.Background(), app, orgID, "redroid")
	require.Nil(t, apiErr)
	// idle and also-idle tie at backlog 1; the lexicographically smaller wins.
	assert.Equal(t, "usera-s-organization/also-idle/device-1", deviceID)

	t.Run("queue query failure is a server error", func(t *testing.T) {
		covRunnerStubQueue(
			t,
			func(context.Context, string) (workflows.MobileDeviceSemaphoreStateView, error) {
				return workflows.MobileDeviceSemaphoreStateView{}, errors.New("temporal down")
			},
		)
		_, apiErr := selectPipelineCIDeviceByType(context.Background(), app, orgID, "redroid")
		require.NotNil(t, apiErr)
		assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
		assert.Equal(t, "failed to query runner queue", apiErr.Reason)
	})

	t.Run("health check failure is a server error", func(t *testing.T) {
		covRunnerStubHealth(t, func(context.Context, string) (bool, error) {
			return false, errors.New("dial failure")
		})
		_, apiErr := selectPipelineCIDeviceByType(
			context.Background(),
			app,
			orgID,
			"android_emulator",
		)
		require.NotNil(t, apiErr)
		assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
		assert.Equal(t, "failed to check runner health", apiErr.Reason)
	})
}

func TestRequireMobileDeviceRunnersOnline(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	createMobileRunnerRecord(t, app, orgID, "multi", "https://multi.example", false)
	runner, err := canonify.Resolve(app, "/usera-s-organization/multi")
	require.NoError(t, err)
	covRunnerCreateDevice(t, app, runner, "d1", "redroid", "")
	covRunnerCreateDevice(t, app, runner, "d2", "android_phone", "")
	createMobileRunnerRecord(t, app, orgID, "nourl", "https://nourl.example", false)
	_, err = app.DB().NewQuery("UPDATE mobile_runners SET ip = '' WHERE name = 'nourl'").Execute()
	require.NoError(t, err)
	noURL, err := canonify.Resolve(app, "/usera-s-organization/nourl")
	require.NoError(t, err)
	covRunnerCreateDevice(t, app, noURL, "d3", "redroid", "")

	testCases := []struct {
		name       string
		devices    []string
		health     func(context.Context, string) (bool, error)
		wantProbes int
		wantStatus int
		wantReason string
	}{
		{
			name:       "devices on the same runner probe it once",
			devices:    []string{"usera-s-organization/multi/d1", "/usera-s-organization/multi/d2"},
			health:     func(context.Context, string) (bool, error) { return true, nil },
			wantProbes: 1,
		},
		{
			name:       "offline runner",
			devices:    []string{"usera-s-organization/multi/d1"},
			health:     func(context.Context, string) (bool, error) { return false, nil },
			wantProbes: 1,
			wantStatus: http.StatusServiceUnavailable,
			wantReason: "device runner is offline",
		},
		{
			name:       "runner without url is offline without probing",
			devices:    []string{"usera-s-organization/nourl/d3"},
			health:     func(context.Context, string) (bool, error) { return true, nil },
			wantStatus: http.StatusServiceUnavailable,
			wantReason: "device runner is offline",
		},
		{
			name:       "health check error",
			devices:    []string{"usera-s-organization/multi/d1"},
			health:     func(context.Context, string) (bool, error) { return false, errors.New("x") },
			wantProbes: 1,
			wantStatus: http.StatusInternalServerError,
			wantReason: "failed to check runner health",
		},
		{
			name:       "identifier of a runner is not a device",
			devices:    []string{"usera-s-organization/multi"},
			health:     func(context.Context, string) (bool, error) { return true, nil },
			wantStatus: http.StatusNotFound,
			wantReason: "mobile_device_not_found",
		},
		{
			name:       "unknown device",
			devices:    []string{"usera-s-organization/multi/ghost"},
			health:     func(context.Context, string) (bool, error) { return true, nil },
			wantStatus: http.StatusNotFound,
			wantReason: "mobile_device_not_found",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			probes := 0
			covRunnerStubHealth(t, func(ctx context.Context, u string) (bool, error) {
				probes++
				return tc.health(ctx, u)
			})
			apiErr := requireMobileDeviceRunnersOnline(context.Background(), app, tc.devices)
			assert.Equal(t, tc.wantProbes, probes)
			if tc.wantStatus == 0 {
				assert.Nil(t, apiErr)
				return
			}
			require.NotNil(t, apiErr)
			assert.Equal(t, tc.wantStatus, apiErr.Code)
			assert.Equal(t, tc.wantReason, apiErr.Reason)
		})
	}
}

func TestCheckRunnerReachableHTTP(t *testing.T) {
	var gotPath string
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer healthy.Close()
	unhealthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer unhealthy.Close()
	closed := httptest.NewServer(http.NotFoundHandler())
	closedURL := closed.URL
	closed.Close()

	testCases := []struct {
		name    string
		url     string
		want    bool
		wantErr bool
	}{
		{name: "200 health is online", url: healthy.URL + "/base/", want: true},
		{name: "non-200 health is offline", url: unhealthy.URL},
		{name: "unreachable runner is offline without error", url: closedURL},
		{name: "unparseable url is an error", url: "http://[::1", wantErr: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			online, err := checkRunnerReachableHTTP(context.Background(), tc.url)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, online)
		})
	}
	assert.Equal(t, "/base/health", gotPath)
}

func TestRewriteStepCIHostRejectsUnusableDocuments(t *testing.T) {
	testCases := []struct {
		name string
		yaml string
	}{
		{name: "blank document", yaml: "  \n"},
		{name: "invalid yaml", yaml: "env: [unterminated"},
		{name: "env is not a mapping", yaml: "env: just-a-string\n"},
		{name: "host is not a string", yaml: "env:\n  host:\n    nested: true\n"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rewritten, ok := rewriteStepCIHost(tc.yaml, "https://new.example")
			assert.False(t, ok)
			assert.Empty(t, rewritten)
		})
	}
}

func TestParsePipelineCIWorkflowRejectsInvalidYAML(t *testing.T) {
	def, apiErr := parsePipelineCIWorkflow("steps: [")
	assert.Nil(t, def)
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)
	assert.Equal(t, "failed to parse pipeline yaml", apiErr.Reason)
}

func TestPipelineCIStepRunnerOnlyPipelines(t *testing.T) {
	const stepRunnerYAML = "name: t\nsteps:\n  - id: m\n    use: mobile-automation\n    with:\n      device_id: o/r/d\n"
	def, apiErr := parsePipelineCIWorkflow(stepRunnerYAML)
	require.Nil(t, apiErr)

	deviceID, hasStep, needsGlobal, apiErr := resolvePipelineCIDeviceID(
		context.Background(), nil, "org", def, pipelineCIBaseRequest{},
	)
	require.Nil(t, apiErr)
	assert.Empty(t, deviceID)
	assert.True(t, hasStep)
	assert.False(t, needsGlobal)

	noMobile, apiErr := parsePipelineCIWorkflow(
		"name: t\nsteps:\n  - id: h\n    use: http-request\n    with:\n      url: x\n",
	)
	require.Nil(t, apiErr)
	got, apiErr := injectPipelineCIGlobalDeviceID("original", noMobile, "o/r/d", false, false)
	require.Nil(t, apiErr)
	assert.Equal(t, "original", got, "a pipeline without mobile steps keeps its yaml")
}

func TestPipelineCIStepRefsIgnoreNonStringPayloads(t *testing.T) {
	const pipelineYAML = "name: t\nsteps:\n" +
		"  - id: a\n    use: use-case-verification-deeplink\n    with:\n      use_case_id:\n        nested: x\n" +
		"  - id: b\n    use: use-case-verification-deeplink\n    with:\n      use_case_id: org/v/b\n"
	def, apiErr := parsePipelineCIWorkflow(pipelineYAML)
	require.Nil(t, apiErr)
	assert.Equal(t,
		[]string{"org/v/b"},
		collectPipelineCIReferences(def, "use-case-verification-deeplink", "use_case_id"),
	)

	got, apiErr := rewritePipelineCIStepRefsYAML(
		pipelineYAML,
		map[string]string{"org/v/b": "org/v/b-tmp"},
		"use-case-verification-deeplink",
		"use_case_id",
	)
	require.Nil(t, apiErr)
	rewritten, apiErr := parsePipelineCIWorkflow(got)
	require.Nil(t, apiErr)
	assert.Equal(t, map[string]any{"nested": "x"}, rewritten.Steps[0].With.Payload["use_case_id"])
	assert.Equal(t, "org/v/b-tmp", rewritten.Steps[1].With.Payload["use_case_id"])
}

func TestCreatePipelineCITempRecordsSkipsRefsWithoutHost(t *testing.T) {
	app := setupPipelineVerifierCIApp(t)
	defer app.Cleanup()
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	ref := createVerifierCIUseCase(t, app, orgID, "Verifier", "No Host")
	record, err := canonify.Resolve(app, ref)
	require.NoError(t, err)
	record.Set("yaml", "version: '1.1'\nenv:\n  body: x\n")
	require.NoError(t, app.Save(record))

	event := &core.RequestEvent{App: app}
	temps, rewriteMap, apiErr := createPipelineCITempRecords(event, pipelineCITempRecordsOptions{
		Refs:           []string{ref},
		Collection:     "use_cases_verifications",
		IdentifierKey:  "use_case_id",
		IdentifierName: "use case verification",
		ResourceDomain: "use_case_verification",
		ResourceName:   "use case verification",
		OwnerID:        orgID,
		CommitSHA:      "abc",
		HostURL:        "https://ci.example",
	})
	require.Nil(t, apiErr)
	assert.Empty(t, temps)
	assert.Empty(t, rewriteMap)

	// Rolling back entries without a persisted record is a no-op.
	rollbackPipelineCITempRecords(
		event,
		[]pipelineCITempRecordResult{{}, {Record: &core.Record{}}},
		"x",
	)
	_, err = canonify.Resolve(app, ref)
	assert.NoError(t, err)
}
