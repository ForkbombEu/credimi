// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/fcaf/walletroles"
	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
)

// fcafSessionLookup fakes a running, Credimi-signed FCAF session of the
// relying-party suite on session's build.
func fcafSessionLookup(
	t *testing.T,
	app core.App,
	session actionauthor.BuildRequest,
) sessionLookup {
	t.Helper()
	lookup := unsignedSessionLookup(app)
	lookup.history = signedSessionHistory(t, app, map[string]any{
		"platform":                  session.Platform,
		"wallet_version_identifier": session.WalletVersionIdentifier,
		"actor_id":                  session.ActorID,
		"mode":                      actionauthor.SessionModeFCAF,
		"fcaf_suite":                actionauthor.FCAFRelyingPartySuite,
	})
	return lookup
}

// testFCAFScope is the repository contract's relying-party scope.
func testFCAFScope(t *testing.T) *actionauthor.FCAFScope {
	t.Helper()
	scope, err := FCAFScope(ActionAuthorSessionRef{
		Mode:      actionauthor.SessionModeFCAF,
		FCAFSuite: actionauthor.FCAFRelyingPartySuite,
	})
	require.NoError(t, err)
	require.NotNil(t, scope)
	return scope
}

// withStarts completes an FCAF plan with candidate prerequisites for the
// roles it starts from.
func withStarts(plan actionauthor.Spec, starts []string) actionauthor.Spec {
	plan.Prerequisites = nil
	for i := range starts {
		plan.Prerequisites = append(plan.Prerequisites, actionauthor.PrerequisiteSpec{
			Candidate: "a" + string(rune('1'+i)),
		})
	}
	return plan
}

// hostRewrite sends every request to host through next.
type hostRewrite struct {
	host string
	next http.RoundTripper
}

func (r hostRewrite) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Host = r.host
	req.Host = r.host
	return r.next.RoundTrip(req)
}

func TestFCAFScope(t *testing.T) {
	scope, err := FCAFScope(ActionAuthorSessionRef{})
	require.NoError(t, err)
	require.Nil(t, scope)

	scope = testFCAFScope(t)
	require.Equal(t, actionauthor.FCAFRelyingPartySuite, scope.Suite)
	require.NotEmpty(t, scope.Contract.Roles)
}

func TestActionAuthorFCAFActivitiesRefuseChatSessions(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, _ := authorSaveFixture(t)
	caller := authorTestCaller()
	// A faked session history is read once: each activity gets its own.
	lookup := func() authorSessionLookup { return buildSessionLookup(t, app, session).lookup() }

	list := NewListActionAuthorFCAFRolesActivity(app)
	list.authorSessionLookup = lookup()
	describe := NewDescribeActionAuthorFCAFRoleActivity(app)
	describe.authorSessionLookup = lookup()
	check := NewCheckActionAuthorFCAFEvidenceActivity(app)
	check.authorSessionLookup = lookup()

	for name, run := range map[string]func() error{
		"list": func() error {
			_, err := list.list(t.Context(), caller)
			return err
		},
		"describe": func() error {
			_, err := describe.describe(t.Context(), caller, DescribeActionAuthorFCAFRoleInput{
				Role: "fcaf-reset-and-onboard",
			})
			return err
		},
		"evidence": func() error {
			_, err := check.check(t.Context(), caller, CheckActionAuthorFCAFEvidenceInput{})
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := run()
			requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
			require.ErrorContains(t, err, "this session does not author FCAF actions")
		})
	}
}

func TestActionAuthorFCAFActivitiesRejectCallersOutsideASession(t *testing.T) {
	for name, a := range map[string]workflowengine.ExecutableActivity{
		"list":     NewListActionAuthorFCAFRolesActivity(nil),
		"describe": NewDescribeActionAuthorFCAFRoleActivity(nil),
		"evidence": NewCheckActionAuthorFCAFEvidenceActivity(nil),
	} {
		// The test environment's caller is a workflow of another type.
		_, err := executeActivity(t, a, map[string]any{})
		requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
		require.ErrorContains(t, err, "caller is not an action author session agent", name)

		_, err = executeActivity(t, a, "not a request")
		requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, false)
	}
}

func TestListActionAuthorFCAFRoles(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, _ := authorSaveFixture(t)
	a := NewListActionAuthorFCAFRolesActivity(app)
	a.authorSessionLookup = fcafSessionLookup(t, app, session).lookup()

	output, err := a.list(t.Context(), authorTestCaller())
	require.NoError(t, err)
	scope := testFCAFScope(t)
	require.Equal(t, actionauthor.FCAFRelyingPartySuite, output.Suite)
	require.Equal(t, scope.Contract.Rules, output.Rules)
	require.Nil(t, output.Profile)
	require.Equal(t, []string{}, output.Warnings)
	require.NotEmpty(t, output.Roles)
	require.Equal(t, scope.Contract.Injected(walletroles.InjectedReset).Name, output.Roles[0].Role)
	require.Equal(
		t,
		scope.Contract.Injected(walletroles.InjectedIssuance).Name,
		output.Roles[1].Role,
	)
	for _, role := range output.Roles {
		require.Equal(t, actionauthor.FCAFMissing, role.Status, role.Role)
	}

	// A profile whose PID issuer is jwt-proof misses FCAF's key attestation.
	reset := scope.Contract.Injected(walletroles.InjectedReset).Name
	action := seedAuthorAction(t, app, session, reset, "appId: eu.wallet")
	for issuer, warnings := range map[string][]string{
		walletroles.PIDIssuerJWTProof:       {actionauthor.FCAFKeyAttestationFinding},
		walletroles.PIDIssuerKeyAttestation: {},
	} {
		profile := map[string]any{"pin": "123456", "pid_issuer": issuer}
		recordProvenance(t, app, action, actionauthor.Provenance{
			FCAFRole:    reset,
			FCAFProfile: profile,
		})
		a.authorSessionLookup = fcafSessionLookup(t, app, session).lookup()
		output, err = a.list(t.Context(), authorTestCaller())
		require.NoError(t, err, issuer)
		require.Equal(t, profile, output.Profile, issuer)
		require.Equal(t, warnings, output.Warnings, issuer)
	}
}

func TestDescribeActionAuthorFCAFRole(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, _ := authorSaveFixture(t)
	caller := authorTestCaller()
	describe := func(role string) (DescribeActionAuthorFCAFRoleOutput, error) {
		a := NewDescribeActionAuthorFCAFRoleActivity(app)
		a.authorSessionLookup = fcafSessionLookup(t, app, session).lookup()
		return a.describe(t.Context(), caller, DescribeActionAuthorFCAFRoleInput{Role: role})
	}

	rejected, err := describe("fcaf-expect-request-rejected")
	require.NoError(t, err)
	require.Equal(t, "fcaf-expect-request-rejected", rejected.Role.Name)
	require.NotEmpty(t, rejected.Role.ReferenceCode)
	require.Equal(t, rejected.Role.SHA256(), rejected.RoleSHA256)
	require.Equal(t, []string{"fcaf-reset-and-onboard", "fcaf-issue-pid"}, rejected.StartsFrom)
	require.Nil(t, rejected.Plan)
	require.Equal(t, "com.android.chrome", rejected.BrowserAppID)
	require.NotEmpty(t, rejected.Fixtures)
	require.LessOrEqual(t, len(rejected.Fixtures), fcafDescribeFixtures)
	first := rejected.Fixtures[0]
	require.NotEmpty(t, first.TestIDs)
	require.Equal(t, "fcaf-expect-request-rejected", first.Plan.FCAFRole)
	require.Len(t, first.Plan.Parameters, 1)
	source := first.Plan.Parameters[0].Source
	require.Equal(t, actionauthor.SourceHTTP, source.Kind)
	require.Equal(t, http.MethodPost, source.Request.Method)
	require.Equal(t, "https://capture-wallet.credimi.io/openid4vp/sessions", source.Request.URL)
	require.Equal(t, map[string]string{"session_id": "session_id"}, source.Captures)
	require.Equal(t, actionauthor.SourceSpec{
		Kind: actionauthor.SourceHTTP,
		Request: httpSourceRequest(
			walletroles.PIDIssuance(walletroles.PIDIssuerKeyAttestation, "mdoc"),
		),
		ValuePath: "deeplink",
	}, rejected.PIDIssuance[walletroles.PIDIssuerKeyAttestation]["mdoc"])
	require.ElementsMatch(
		t,
		walletroles.PIDIssuers,
		slices.Collect(maps.Keys(rejected.PIDIssuance)),
	)
	for _, formats := range rejected.PIDIssuance {
		require.ElementsMatch(t, []string{"sd-jwt", "mdoc"}, slices.Collect(maps.Keys(formats)))
	}
	jwtProof := rejected.PIDIssuance[walletroles.PIDIssuerJWTProof]["sd-jwt"]
	require.Equal(t, http.MethodPost, jwtProof.Request.Method)
	require.Equal(t, map[string]any{
		"issuer_configuration_id":     "eu-pid-jwt-proof-only",
		"credential_configuration_id": "urn:eu.europa.ec.eudi:pid:1.sd-jwt.jwt-proof",
	}, jwtProof.Request.Body)

	reset, err := describe("fcaf-reset-and-onboard")
	require.NoError(t, err)
	require.NotNil(t, reset.Plan)
	require.Equal(t, "fcaf-reset-and-onboard", reset.Plan.FCAFRole)
	require.Empty(t, reset.Fixtures)
	require.NotNil(t, reset.Fixtures)
	require.Empty(t, reset.StartsFrom)
	require.NotNil(t, reset.StartsFrom)

	pid, err := describe("fcaf-issue-pid")
	require.NoError(t, err)
	ids := make([]string, 0, len(pid.Fixtures))
	issuers := make([]string, 0, len(pid.Fixtures))
	for _, fixture := range pid.Fixtures {
		ids = append(ids, fixture.ID)
		issuers = append(issuers, fixture.PIDIssuer)
	}
	require.Equal(
		t,
		[]string{"pid-sd-jwt", "pid-mdoc", "pid-jwt-proof-sd-jwt", "pid-jwt-proof-mdoc"},
		ids,
	)
	require.Equal(t, []string{
		walletroles.PIDIssuerKeyAttestation,
		walletroles.PIDIssuerKeyAttestation,
		walletroles.PIDIssuerJWTProof,
		walletroles.PIDIssuerJWTProof,
	}, issuers)
	for _, fixture := range rejected.Fixtures {
		require.Empty(t, fixture.PIDIssuer, fixture.ID)
	}

	for name, tc := range map[string]struct {
		role string
		want string
	}{
		"unknown role":      {role: "fcaf-unknown", want: "fcaf_role: fcaf-unknown is not an FCAF role"},
		"outside the suite": {role: "fcaf-observe-wallet", want: "is not part of the wallet_solution/relying_party suite"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := describe(tc.role)
			requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

// Every plan Describe builds for a relying-party role, completed with
// candidates for the roles it starts from, passes the contract check.
func TestDescribeFCAFRolePlansPassTheContract(t *testing.T) {
	scope := testFCAFScope(t)
	for _, role := range scope.Contract.Roles {
		if !slices.Contains(role.Suites, scope.Suite) {
			continue
		}
		t.Run(role.Name, func(t *testing.T) {
			described, err := describeFCAFRole(scope, role.Name)
			require.NoError(t, err)
			plans := []actionauthor.Spec{}
			if described.Plan != nil {
				plans = append(plans, *described.Plan)
			}
			for _, fixture := range described.Fixtures {
				plans = append(plans, fixture.Plan)
			}
			require.NotEmpty(t, plans)
			for _, plan := range plans {
				plan = withStarts(plan, described.StartsFrom)
				require.NoError(t, plan.Normalize())
				_, err := actionauthor.CheckFCAFPlan(scope, &plan)
				require.NoError(t, err)
			}
		})
	}
}

func TestCheckActionAuthorFCAFEvidence(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, _ := authorSaveFixture(t)
	scope := testFCAFScope(t)
	described, err := describeFCAFRole(scope, "fcaf-expect-request-rejected")
	require.NoError(t, err)
	fixture := described.Fixtures[0]
	plan := withStarts(fixture.Plan, described.StartsFrom)

	var fetched atomic.Value
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetched.Store(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/missing") {
			w.WriteHeader(http.StatusNotFound)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"session_id": "s1", "status": "created"})
	}))
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	check := func(payload CheckActionAuthorFCAFEvidenceInput) (CheckActionAuthorFCAFEvidenceOutput, error) {
		a := NewCheckActionAuthorFCAFEvidenceActivity(app)
		a.authorSessionLookup = fcafSessionLookup(t, app, session).lookup()
		a.transport = hostRewrite{host: serverURL.Host, next: server.Client().Transport}
		return a.check(t.Context(), authorTestCaller(), payload)
	}
	screenshotProblems := func(output CheckActionAuthorFCAFEvidenceOutput) []FCAFEvidenceAssertion {
		var problems []FCAFEvidenceAssertion
		for _, test := range output.Tests {
			require.NotNil(t, test.Problems)
			for _, problem := range test.Problems {
				if problem.Validator == "evidence.screenshot_named" {
					problems = append(problems, problem)
				}
			}
		}
		return problems
	}
	valid := CheckActionAuthorFCAFEvidenceInput{
		Plan:        plan,
		Fixture:     fixture.ID,
		Captures:    map[string]string{"session_id": "s1"},
		Screenshots: []string{"request-refused"},
	}

	refused, err := check(valid)
	require.NoError(t, err)
	require.Equal(t, "GET /openid4vp/sessions/s1", fetched.Load())
	require.NotEmpty(t, refused.Tests)
	require.Contains(t, []string{"passed", "failed"}, refused.Status)
	require.Empty(t, screenshotProblems(refused))

	other := valid
	other.Screenshots = []string{"other"}
	wrong, err := check(other)
	require.NoError(t, err)
	problems := screenshotProblems(wrong)
	require.NotEmpty(t, problems)
	for _, problem := range problems {
		require.Equal(t, "fail", problem.Status)
	}
	require.Equal(t, "failed", wrong.Status)

	chatPlan := actionauthor.Spec{Slot: "onboarding"}
	for name, tc := range map[string]struct {
		mutate func(*CheckActionAuthorFCAFEvidenceInput)
		want   string
	}{
		"unknown fixture": {
			mutate: func(in *CheckActionAuthorFCAFEvidenceInput) { in.Fixture = "missing#step" },
			want:   "fixture missing#step is not one of " + fixture.ID,
		},
		"missing session id": {
			mutate: func(in *CheckActionAuthorFCAFEvidenceInput) { in.Captures = nil },
			want:   "captures.session_id is required",
		},
		"session id outside its shape": {
			mutate: func(in *CheckActionAuthorFCAFEvidenceInput) {
				in.Captures = map[string]string{"session_id": "../x"}
			},
			want: "captures.session_id must be 1-128 letters, digits, '-' or '_'",
		},
		"fetch of a missing session": {
			mutate: func(in *CheckActionAuthorFCAFEvidenceInput) {
				in.Captures = map[string]string{"session_id": "missing"}
			},
			want: "answered HTTP 404",
		},
		"bad screenshot name": {
			mutate: func(in *CheckActionAuthorFCAFEvidenceInput) { in.Screenshots = []string{"a b"} },
			want:   `screenshots: "a b" is not a screenshot name`,
		},
		"too many screenshots": {
			mutate: func(in *CheckActionAuthorFCAFEvidenceInput) {
				in.Screenshots = make([]string, fcafEvidenceScreenshots+1)
			},
			want: "screenshots: at most 64 names",
		},
		"plan without a role": {
			mutate: func(in *CheckActionAuthorFCAFEvidenceInput) { in.Plan = chatPlan },
			want:   "the plan has no fcaf_role",
		},
		"plan against the contract": {
			mutate: func(in *CheckActionAuthorFCAFEvidenceInput) {
				in.Plan.Prerequisites = in.Plan.Prerequisites[:1]
			},
			want: "fcaf_role: fcaf-expect-request-rejected starts from",
		},
	} {
		t.Run(name, func(t *testing.T) {
			payload := valid
			tc.mutate(&payload)
			_, err := check(payload)
			requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

// Save refuses an FCAF profile other than the one the user approved.
func TestSaveAuthoredWalletActionBindsTheFCAFProfile(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, namespace := authorSaveFixture(t)
	build, err := actionauthor.Resolve(app, namespace, session)
	require.NoError(t, err)
	selection, err := actionauthor.Select(app, namespace, session)
	require.NoError(t, err)
	caller := authorTestCaller()
	scope := testFCAFScope(t)
	role := scope.Contract.Injected(walletroles.InjectedReset)
	code := strings.ReplaceAll(
		role.ReferenceCode,
		scope.Contract.Reference.AppIDs[walletroles.AppWallet],
		"eu.wallet",
	)
	plan := actionauthor.Spec{
		Slot:     "onboarding",
		Outcome:  actionauthor.OutcomeSuccess,
		FCAFRole: role.Name,
	}
	normalized, err := SessionPlan(app, selection, plan, scope)
	require.NoError(t, err)
	profile := map[string]any{"pin": "123456"}
	approval := func(profile map[string]any) string {
		token, err := actionauthor.SignApproval(app, actionauthor.Approval{
			AuthorWorkflowID: caller.WorkflowID,
			CandidateSHA256:  actionauthor.CodeSHA256(code),
			ApproverID:       session.ActorID,
			ApprovedAt:       time.Now().Unix(),
			PlanSHA256:       actionauthor.PlanSHA256(normalized),
			Key:              "a1",
			ProfileSHA256:    actionauthor.ProfileSHA256(profile),
		})
		require.NoError(t, err)
		return token
	}
	payload := actionauthor.SaveRequest{
		Name:            role.Name,
		Code:            code,
		CandidateSHA256: actionauthor.CodeSHA256(code),
		InstallerSHA256: build.InstallerSHA256,
		PackageID:       "eu.wallet",
		ReplayCount:     1,
		Spec:            plan,
		BatchKey:        "a1",
		FCAFProfile:     profile,
	}
	save := func(payload actionauthor.SaveRequest) (actionauthor.Receipt, error) {
		return fcafSessionLookup(t, app, session).activity(t, app).
			save(t.Context(), caller, payload)
	}

	for name, token := range map[string]string{
		"no profile approved": approval(nil),
		"another profile":     approval(map[string]any{"pin": "000000"}),
	} {
		refused := payload
		refused.Approval = token
		_, err := save(refused)
		requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
		require.ErrorContains(t, err, "approval is for another profile", name)
	}

	payload.Approval = approval(profile)
	receipt, err := save(payload)
	require.NoError(t, err)
	require.Equal(t, profile, receipt.Provenance.FCAFProfile)
}
