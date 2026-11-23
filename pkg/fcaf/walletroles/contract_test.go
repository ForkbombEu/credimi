// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package walletroles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const fixtureContract = `version: 1
reference:
  manifest: wallet-actions.yaml
  app_ids:
    wallet: org.example.wallet
    browser: com.android.chrome
rules: []
roles:
  - name: reset
    summary: Resets the wallet.
    suites: [wallet_solution/relying_party]
    outcome: strict-success
    fails_when: the wallet does not reach Home
    parameters: []
    header_app: wallet
    launch_apps: [wallet]
    presents: 0
    injected: reset
    silent: true
    diagnostic_name_budget: 10
    screenshots: []
  - name: issue
    summary: Accepts an offer.
    suites: [wallet_solution/relying_party]
    outcome: strict-success
    fails_when: the credential is not stored
    parameters: [deeplink]
    header_app: wallet
    launch_apps: [wallet]
    presents: 0
    issues_credential: true
    injected: issuance
    silent: true
    diagnostic_name_budget: 10
    screenshots: []
`

const fixtureManifest = `organization: example
wallet: wallet
actions:
  - name: reset
    category: onboarding
    file: wallet/reset.yaml
  - name: issue
    category: get-credential-generic
    file: wallet/issue.yaml
`

const fixtureFlow = "appId: org.example.wallet\n---\n- launchApp\n"

func writeFixture(t *testing.T, contract string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		FileName:              contract,
		"wallet-actions.yaml": fixtureManifest,
		"wallet/reset.yaml":   fixtureFlow,
		"wallet/issue.yaml":   fixtureFlow + "- openLink: ${deeplink}\n",
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return root
}

func TestLoadValidContract(t *testing.T) {
	contract, err := Load(writeFixture(t, fixtureContract))
	require.NoError(t, err)
	issue, found := contract.Role("issue")
	require.True(t, found)
	require.Equal(t, "get-credential-generic", issue.Category)
	require.Contains(t, issue.ReferenceCode, "${deeplink}")
	require.Equal(t, "reset", contract.Injected(InjectedReset).Name)
	require.Equal(t, "issue", contract.Injected(InjectedIssuance).Name)
}

const issueRoleStart = "  - name: issue\n"

func TestLoadRejectsInvalidContract(t *testing.T) {
	_, issueRest, _ := strings.Cut(fixtureContract, issueRoleStart)
	issueRole := issueRoleStart + issueRest
	for _, testCase := range []struct {
		name    string
		mutate  func(string) string
		message string
	}{
		{
			name:    "duplicate role name",
			mutate:  func(c string) string { return c + issueRole },
			message: `role "issue"`,
		},
		{
			name: "unknown outcome",
			mutate: func(c string) string {
				return replaceInIssue(c, "outcome: strict-success", "outcome: maybe")
			},
			message: `role "issue"`,
		},
		{
			name: "wait without wait_seconds",
			mutate: func(c string) string {
				return replaceInIssue(c, "outcome: strict-success", "outcome: wait")
			},
			message: `role "issue"`,
		},
		{
			name: "wait_seconds on a non-wait role",
			mutate: func(c string) string {
				return replaceInIssue(
					c,
					"outcome: strict-success",
					"outcome: strict-success\n    wait_seconds: 20",
				)
			},
			message: `role "issue"`,
		},
		{
			name: "silent role with screenshots",
			mutate: func(c string) string {
				return replaceInIssue(
					c,
					"screenshots: []",
					"screenshots:\n      - name: added\n        when: unconditional",
				)
			},
			message: `role "issue"`,
		},
		{
			name: "two reset roles",
			mutate: func(c string) string {
				return replaceInIssue(c, "injected: issuance", "injected: reset")
			},
			message: "injected: reset",
		},
		{
			name: "role missing from the manifest",
			mutate: func(c string) string {
				return c + strings.ReplaceAll(
					strings.ReplaceAll(issueRole, "name: issue", "name: orphan"),
					"    injected: issuance\n    silent: true\n",
					"",
				)
			},
			message: `role "orphan"`,
		},
		{
			name: "unknown key",
			mutate: func(c string) string {
				return replaceInIssue(c, "presents: 0", "presents: 0\n    shares: 1")
			},
			message: "shares",
		},
		{
			name: "gated screenshot without screen",
			mutate: func(c string) string {
				role := strings.ReplaceAll(issueRole, "name: issue", "name: shot")
				role = strings.ReplaceAll(role, "    injected: issuance\n    silent: true\n", "")
				role = strings.ReplaceAll(
					role,
					"screenshots: []",
					"screenshots:\n      - name: added\n        when: gated",
				)
				return c + role
			},
			message: `role "shot"`,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Load(writeFixture(t, testCase.mutate(fixtureContract)))
			require.ErrorContains(t, err, testCase.message)
		})
	}
}

// replaceInIssue replaces the first occurrence of old in the issue role.
func replaceInIssue(contract string, old string, replacement string) string {
	head, role, _ := strings.Cut(contract, issueRoleStart)
	return head + issueRoleStart + strings.Replace(role, old, replacement, 1)
}

func TestScreenshotConditions(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		commands string
		want     map[string][]ScreenshotCondition
	}{
		{
			name:     "unconditional screenshot",
			commands: "- takeScreenshot: launch\n",
			want:     map[string][]ScreenshotCondition{"launch": {ScreenshotUnconditional}},
		},
		{
			name: "optional wait does not gate",
			commands: `- extendedWaitUntil:
    visible: Home
    timeout: 1000
    optional: true
- takeScreenshot:
    path: home
`,
			want: map[string][]ScreenshotCondition{"home": {ScreenshotUnconditional}},
		},
		{
			name:     "strict assertion gates",
			commands: "- assertVisible: Done\n- takeScreenshot: done\n",
			want:     map[string][]ScreenshotCondition{"done": {ScreenshotGated}},
		},
		{
			name: "screenshot inside when",
			commands: `- runFlow:
    when:
      visible: Error
    commands:
      - takeScreenshot: error
`,
			want: map[string][]ScreenshotCondition{"error": {ScreenshotWhileScreen}},
		},
		{
			name: "strict check inside when does not gate later screenshots",
			commands: `- runFlow:
    when:
      visible: Error
    commands:
      - assertVisible: Error
- takeScreenshot: after
`,
			want: map[string][]ScreenshotCondition{"after": {ScreenshotUnconditional}},
		},
		{
			name:     "scalar commands are ignored",
			commands: "- \"stopApp\"\n- takeScreenshot: stopped\n",
			want:     map[string][]ScreenshotCondition{"stopped": {ScreenshotUnconditional}},
		},
		{
			name: "strict check inside repeat does not gate later screenshots",
			commands: `- repeat:
    times: 2
    commands:
      - assertVisible: Done
- takeScreenshot: after
`,
			want: map[string][]ScreenshotCondition{"after": {ScreenshotUnconditional}},
		},
		{
			name: "optional yes does not gate",
			commands: `- assertVisible:
    text: Done
    optional: yes
- takeScreenshot: after
`,
			want: map[string][]ScreenshotCondition{"after": {ScreenshotUnconditional}},
		},
		{
			name: "quoted optional false does not gate",
			commands: `- assertVisible:
    text: Done
    optional: "false"
- takeScreenshot: after
`,
			want: map[string][]ScreenshotCondition{"after": {ScreenshotUnconditional}},
		},
		{
			name: "plain optional false gates",
			commands: `- assertVisible:
    text: Done
    optional: false
- takeScreenshot: after
`,
			want: map[string][]ScreenshotCondition{"after": {ScreenshotGated}},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := ScreenshotConditions("appId: org.example.wallet\n---\n" + testCase.commands)
			require.NoError(t, err)
			require.Equal(t, testCase.want, got)
		})
	}

	t.Run("single document flow", func(t *testing.T) {
		_, err := ScreenshotConditions("- takeScreenshot: launch\n")
		require.Error(t, err)
	})
}
