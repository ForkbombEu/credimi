// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	codeTestWallet = "com.example.wallet"
	codeTestTail   = "- assertVisible: \"Welcome\"\n- takeScreenshot: done\n"
)

func TestAllowedAppIDs(t *testing.T) {
	assert.Equal(t, []string{codeTestWallet}, AllowedAppIDs(codeTestWallet, false))
	assert.Equal(
		t,
		[]string{codeTestWallet, "com.android.chrome"},
		AllowedAppIDs(codeTestWallet, true),
	)
}

func TestValidateCode(t *testing.T) {
	wallet := AllowedAppIDs(codeTestWallet, false)
	browser := AllowedAppIDs(codeTestWallet, true)
	header := "appId: com.example.wallet\n---\n"
	lastMessage := "the last command before the final takeScreenshot must assert the outcome: " +
		"`assertVisible`, or `extendedWaitUntil` with `visible`"

	cases := []struct {
		name       string
		code       string
		appIDs     []string
		parameters []string
		want       []string
	}{
		{
			name: "valid onboarding action",
			code: `appId: com.example.wallet
name: onboarding
tags:
  - onboarding
---
- launchApp:
    clearState: true
- tapOn: "Get started"
- inputText: "123456"
- runFlow:
    when:
      visible: "Allow notifications"
    commands:
      - tapOn: "Allow"
- assertVisible: "Welcome"
- takeScreenshot: onboarded
`,
			appIDs: wallet,
		},
		{
			name: "valid browser deeplink action",
			code: `appId: com.android.chrome
---
- launchApp:
    appId: com.android.chrome
- openLink:
    link: ${deeplink}
- retry:
    maxRetries: 3
    commands:
      - tapOn: "Accept"
- launchApp: com.example.wallet
- extendedWaitUntil:
    visible: "Credential added"
    timeout: 10000
- takeScreenshot:
    path: credential-ok
`,
			appIDs:     browser,
			parameters: []string{"deeplink"},
		},
		{
			name:   "invalid YAML",
			code:   "appId: [\n",
			appIDs: wallet,
			want:   []string{"invalid YAML: yaml: line 1: did not find expected node content"},
		},
		{
			name:   "one document",
			code:   "- assertVisible: x\n",
			appIDs: wallet,
			want:   []string{"expected two YAML documents (header, `---`, command list), found 1"},
		},
		{
			name:   "document shapes",
			code:   "- appId\n---\nback: true\n",
			appIDs: wallet,
			want: []string{
				"the first document must be the header mapping (appId, ...)",
				"the second document must be a non-empty list of commands",
				"header appId must be 'com.example.wallet', found None",
				"add at least one `takeScreenshot` (after the final assertion)",
				lastMessage,
			},
		},
		{
			name:   "header keys",
			code:   "appId: com.example.wallet\nonFlowStart: x\nextra: y\n---\n" + codeTestTail,
			appIDs: wallet,
			want: []string{
				"header keys not allowed: ['extra', 'onFlowStart']; allowed: ['appId', 'env', 'name', 'tags']",
			},
		},
		{
			name:   "header appId",
			code:   "appId: com.other\n---\n" + codeTestTail,
			appIDs: browser,
			want: []string{
				"header appId must be one of ['com.example.wallet', 'com.android.chrome'], found 'com.other'",
			},
		},
		{
			name:       "header env sets a parameter",
			code:       "appId: com.example.wallet\nenv:\n  deeplink: x\n---\n" + codeTestTail,
			appIDs:     wallet,
			parameters: []string{"deeplink"},
			want:       []string{"header env must not set the session parameter 'deeplink'"},
		},
		{
			name:   "command not allowed, nested",
			code:   header + "- repeat:\n    times: 2\n    commands:\n      - evalScript: x\n" + codeTestTail,
			appIDs: wallet,
			want:   []string{"commands[0].repeat.commands[0]: command `evalScript` is not allowed"},
		},
		{
			name:   "not a command",
			code:   header + "- 42\n" + codeTestTail,
			appIDs: wallet,
			want:   []string{"commands[0]: not a command (use `- name` or `- name: argument`)"},
		},
		{
			name:   "runFlow file",
			code:   header + "- runFlow:\n    file: other.yaml\n" + codeTestTail,
			appIDs: wallet,
			want: []string{
				"commands[0] runFlow: `file` is not allowed; use inline `commands`",
				"commands[0] runFlow: needs a non-empty inline `commands` list",
			},
		},
		{
			name:   "retry file string",
			code:   header + "- retry: other.yaml\n" + codeTestTail,
			appIDs: wallet,
			want:   []string{"commands[0] retry: needs a mapping with inline `commands`"},
		},
		{
			name: "foreign apps",
			code: header + "- launchApp:\n    appId: com.other\n- launchApp: com.evil\n" +
				"- setPermissions:\n    appId: com.other\n    permissions:\n      all: allow\n" + codeTestTail,
			appIDs: wallet,
			want: []string{
				"commands[0]: launchApp targets 'com.other', not the app under test",
				"commands[1]: launchApp targets 'com.evil', not the app under test",
				"commands[2]: setPermissions targets 'com.other', not the app under test",
			},
		},
		{
			name:       "openLink literal",
			code:       header + "- openLink: https://example.org\n- openLink:\n    link: ${b}x\n" + codeTestTail,
			appIDs:     wallet,
			parameters: []string{"b", "a"},
			want: []string{
				"commands[0]: openLink must open a session parameter: write link: ${<one of a, b>}",
				"commands[1]: openLink must open a session parameter: write link: ${<one of a, b>}",
			},
		},
		{
			name:   "openLink without parameters",
			code:   header + "- openLink: https://example.org\n" + codeTestTail,
			appIDs: wallet,
			want: []string{
				"commands[0]: openLink is not allowed: this session has no parameters",
			},
		},
		{
			name: "substitutions",
			code: "appId: com.example.wallet\nname: ${HOME}\n---\n" +
				"- inputText: ${a}\n- tapOn:\n    ${output.x}: y\n- inputText: ${HOME} ${a\n" + codeTestTail,
			appIDs:     wallet,
			parameters: []string{"a"},
			want: []string{
				"substitutions are not allowed: ['${HOME}', '${output.x}', '${a']; write literal values or " +
					"a session parameter (Maestro must not evaluate expressions or read the host environment)",
			},
		},
		{
			name:   "no screenshot",
			code:   header + "- assertVisible: x\n",
			appIDs: wallet,
			want:   []string{"add at least one `takeScreenshot` (after the final assertion)"},
		},
		{
			name: "screenshot names",
			code: header + "- assertVisible: x\n- takeScreenshot: name-longer-than-15\n" +
				"- takeScreenshot:\n    path: a/b\n- takeScreenshot\n",
			appIDs: wallet,
			want: []string{
				"commands[1]: screenshot names must be 1-15 letters, digits, '-' or '_'",
				"commands[2]: screenshot names must be 1-15 letters, digits, '-' or '_'",
				"commands[3]: screenshot names must be 1-15 letters, digits, '-' or '_'",
			},
		},
		{
			name:   "last command is not an assertion",
			code:   header + "- assertVisible: x\n- tapOn: y\n- takeScreenshot: done\n",
			appIDs: wallet,
			want:   []string{lastMessage},
		},
		{
			name:   "extendedWaitUntil without visible",
			code:   header + "- extendedWaitUntil:\n    notVisible: x\n- takeScreenshot: done\n",
			appIDs: wallet,
			want:   []string{lastMessage},
		},
		{
			name:   "duplicate messages",
			code:   "appId: ${x}\n1: a\n2: b\n---\n- tapOn: ${x}\n" + codeTestTail,
			appIDs: wallet,
			want: []string{
				"substitutions are not allowed: ['${x}']; write literal values or a session " +
					"parameter (Maestro must not evaluate expressions or read the host environment)",
				"header keys must be strings",
				"header appId must be 'com.example.wallet', found '${x}'",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ValidateCode(tc.code, tc.appIDs, tc.parameters))
		})
	}
}
