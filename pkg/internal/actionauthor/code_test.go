// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	codeTestWallet = "com.example.wallet"
	codeTestTail   = "- assertVisible: \"Welcome\"\n- takeScreenshot: done\n"
)

func TestAllowedAppIDs(t *testing.T) {
	assert.Equal(
		t,
		[]string{codeTestWallet},
		AllowedAppIDs(codeTestWallet, "onboarding", PlatformAndroid),
	)
	assert.Equal(t,
		[]string{codeTestWallet, "com.android.vending"},
		AllowedAppIDs(codeTestWallet, "install-app", PlatformAndroid))
	assert.Equal(t,
		[]string{codeTestWallet},
		AllowedAppIDs(codeTestWallet, "install-app", PlatformIOS))
}

func TestStoreLinks(t *testing.T) {
	play := "https://play.google.com/store/apps/details?id=" + codeTestWallet
	market := "market://details?id=" + codeTestWallet
	assert.Nil(t, StoreLinks("onboarding", PlatformAndroid, codeTestWallet, play))
	assert.Equal(
		t,
		[]string{market, play},
		StoreLinks("install-app", PlatformAndroid, codeTestWallet, ""),
	)
	assert.Equal(
		t,
		[]string{market, play},
		StoreLinks("install-app", PlatformAndroid, codeTestWallet, play),
	)
	assert.Equal(t,
		[]string{market, play, play + "&hl=en"},
		StoreLinks("install-app", PlatformAndroid, codeTestWallet, play+"&hl=en"))
	assert.Nil(t, StoreLinks("install-app", PlatformIOS, codeTestWallet, play))
}

func TestValidateCode(t *testing.T) {
	wallet := AllowedAppIDs(codeTestWallet, "onboarding", PlatformAndroid)
	installApp := AllowedAppIDs(codeTestWallet, "install-app", PlatformAndroid)
	storeLinks := StoreLinks("install-app", PlatformAndroid, codeTestWallet, "")
	storeMessage := "openLink must open a session parameter or the wallet's store page: " +
		"https://play.google.com/store/apps/details?id=com.example.wallet, " +
		"market://details?id=com.example.wallet"
	header := "appId: com.example.wallet\n---\n"
	lastMessage := "the last command other than takeScreenshot must assert the outcome: " +
		"`assertVisible`, or `extendedWaitUntil` with `visible`"

	waitMessage := "the last command other than takeScreenshot must wait 240 s: " +
		"extendedWaitUntil with optional: true and timeout: 240000"
	waitTail := "- takeScreenshot: done\n"
	staleMessage := func(selector string) string {
		return "the final assertion checks \"" + selector + "\", which the action already " +
			"reached before its last interaction; assert the screen the outcome shows"
	}
	shareInformation := "- launchApp\n- extendedWaitUntil:\n    visible: \"Share information\"\n" +
		"    timeout: 30000\n- tapOn: \"CONFIRM\"\n"

	cases := []struct {
		name        string
		code        string
		appIDs      []string
		parameters  []string
		storeLinks  []string
		waitSeconds int
		want        []string
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
			name: "valid web deeplink action",
			code: `appId: com.example.wallet
---
- launchApp
- openLink:
    link: ${deeplink}
    browser: true
- retry:
    maxRetries: 3
    commands:
      - tapOn: "Accept"
- launchApp: com.example.wallet
- extendedWaitUntil:
    visible: "Credential added"
    timeout: 10000
- takeScreenshot:
    path: fcaf-engagement-haip-vp-invoked
`,
			appIDs:     wallet,
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
			code:   "appId: com.android.chrome\n---\n" + codeTestTail,
			appIDs: wallet,
			want: []string{
				"header appId must be 'com.example.wallet', found 'com.android.chrome'",
			},
		},
		{
			name: "header env sets parameters",
			code: "appId: com.example.wallet\nenv:\n  deeplink: x\n  HOME: y\n  code: z\n---\n" +
				codeTestTail,
			appIDs:     wallet,
			parameters: []string{"deeplink", "code"},
			want: []string{
				"header env must not set session parameters: ['code', 'deeplink']",
			},
		},
		{
			name:   "header appId is not a string",
			code:   "appId: [com.example.wallet]\n---\n" + codeTestTail,
			appIDs: wallet,
			want:   []string{"appId must be a string"},
		},
		{
			name: "command appId is not a string",
			code: header + "- launchApp:\n    appId: {a: b}\n- launchApp: [x]\n" +
				"- setPermissions:\n    appId: 7\n- launchApp\n- launchApp:\n    clearState: true\n" +
				codeTestTail,
			appIDs: wallet,
			want: []string{
				"commands[0]: appId must be a string",
				"commands[1]: appId must be a string",
				"commands[2]: appId must be a string",
			},
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
				"commands[0]: openLink must open a session parameter: write link: ${NAME}, with NAME one of a, b",
				"commands[1]: openLink must open a session parameter: write link: ${NAME}, with NAME one of a, b",
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
			name: "install-app store page",
			code: header + "- openLink:\n    link: \"market://details?id=com.example.wallet\"\n" +
				"    autoVerify: false\n" +
				"- openLink: https://play.google.com/store/apps/details?id=com.example.wallet\n" +
				"- launchApp: com.android.vending\n" +
				"- runFlow:\n    when:\n      visible: \"Install\"\n    commands:\n      - tapOn: \"Install\"\n" +
				"- extendedWaitUntil:\n    visible: \"Open\"\n    timeout: 300000\n" +
				"- takeScreenshot: installed\n",
			appIDs:     installApp,
			storeLinks: storeLinks,
		},
		{
			name: "install-app other link",
			code: header + "- openLink: https://example.org\n- openLink:\n    link: ${a}\n" +
				"- openLink: market://details?id=com.other\n" + codeTestTail,
			appIDs:     installApp,
			parameters: []string{"a"},
			storeLinks: storeLinks,
			want: []string{
				"commands[0]: " + storeMessage,
				"commands[2]: " + storeMessage,
			},
		},
		{
			name:   "store page outside install-app",
			code:   header + "- openLink: \"market://details?id=com.example.wallet\"\n" + codeTestTail,
			appIDs: wallet,
			want: []string{
				"commands[0]: openLink is not allowed: this session has no parameters",
			},
		},
		{
			name:       "store page outside install-app with parameters",
			code:       header + "- openLink: \"market://details?id=com.example.wallet\"\n" + codeTestTail,
			appIDs:     wallet,
			parameters: []string{"a"},
			want: []string{
				"commands[0]: openLink must open a session parameter: write link: ${a}",
			},
		},
		{
			name:   "store app outside install-app",
			code:   header + "- launchApp: com.android.vending\n" + codeTestTail,
			appIDs: wallet,
			want: []string{
				"commands[0]: launchApp targets 'com.android.vending', not the app under test",
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
					"session parameters (Maestro must not evaluate expressions or read the host environment)",
			},
		},
		{
			name:   "no screenshot",
			code:   header + "- launchApp\n- assertVisible: \"Home\"\n",
			appIDs: wallet,
		},
		{
			name: "screenshot names",
			code: header + "- assertVisible: x\n- takeScreenshot: " + strings.Repeat(
				"a",
				65,
			) + "\n" +
				"- takeScreenshot:\n    path: a/b\n- takeScreenshot\n",
			appIDs: wallet,
			want: []string{
				"commands[1]: screenshot names must be 1-64 letters, digits, '-' or '_', " +
					"found '" + strings.Repeat("a", 65) + "'",
				"commands[2]: screenshot names must be 1-64 letters, digits, '-' or '_', found 'a/b'",
				"commands[3]: screenshot names must be 1-64 letters, digits, '-' or '_', found None",
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
				"substitutions are not allowed: ['${x}']; write literal values or session " +
					"parameters (Maestro must not evaluate expressions or read the host environment)",
				"header keys must be strings",
				"header appId must be 'com.example.wallet', found '${x}'",
			},
		},
		{
			name: "duplicate runFlow commands",
			code: header + "- runFlow:\n    commands:\n      - tapOn: a\n    commands:\n" +
				"      - evalScript: x\n" + codeTestTail,
			appIDs: wallet,
			want:   []string{"duplicate key 'commands'"},
		},
		{
			name: "duplicate openLink link",
			code: header + "- openLink:\n    link: ${a}\n    link: https://evil.example\n" +
				codeTestTail,
			appIDs:     wallet,
			parameters: []string{"a"},
			want:       []string{"duplicate key 'link'"},
		},
		{
			name: "duplicate header keys, in order, once each",
			code: "appId: com.example.wallet\nappId: com.other\nname: a\nname: b\n" +
				"appId: c\n---\n" + codeTestTail,
			appIDs: wallet,
			want:   []string{"duplicate key 'appId'", "duplicate key 'name'"},
		},
		{
			name: "merge key",
			code: "appId: com.example.wallet\nenv: &e\n  x: y\n---\n- tapOn:\n    <<: *e\n" +
				"    text: a\n- tapOn:\n    <<: {id: b}\n" + codeTestTail,
			appIDs: wallet,
			want:   []string{"merge keys are not allowed"},
		},
		{
			name:   "quoted merge key is a string",
			code:   header + "- tapOn:\n    \"<<\": a\n" + codeTestTail,
			appIDs: wallet,
		},
		{
			name: "valid wait action",
			code: header + "- launchApp\n- extendedWaitUntil:\n    visible: \"credimi-never\"\n" +
				"    optional: true\n    timeout: 240000\n" + waitTail,
			appIDs:      wallet,
			waitSeconds: 240,
		},
		{
			name: "wait on notVisible",
			code: header + "- launchApp\n- extendedWaitUntil:\n    notVisible: \"credimi-never\"\n" +
				"    optional: true\n    timeout: 240000\n" + waitTail,
			appIDs:      wallet,
			waitSeconds: 240,
			want:        []string{waitMessage},
		},
		{
			name: "wait without optional",
			code: header + "- launchApp\n- extendedWaitUntil:\n    visible: \"credimi-never\"\n" +
				"    timeout: 240000\n" + waitTail,
			appIDs:      wallet,
			waitSeconds: 240,
			want:        []string{waitMessage},
		},
		{
			name: "wait with a string optional",
			code: header + "- launchApp\n- extendedWaitUntil:\n    visible: \"credimi-never\"\n" +
				"    optional: \"true\"\n    timeout: 240000\n" + waitTail,
			appIDs:      wallet,
			waitSeconds: 240,
			want:        []string{waitMessage},
		},
		{
			name: "wait with a YAML 1.1 optional",
			code: header + "- launchApp\n- extendedWaitUntil:\n    visible: \"credimi-never\"\n" +
				"    optional: yes\n    timeout: 240000\n" + waitTail,
			appIDs:      wallet,
			waitSeconds: 240,
			want:        []string{waitMessage},
		},
		{
			name: "wait with a capitalized optional",
			code: header + "- launchApp\n- extendedWaitUntil:\n    visible: \"credimi-never\"\n" +
				"    optional: True\n    timeout: 240000\n" + waitTail,
			appIDs:      wallet,
			waitSeconds: 240,
			want:        []string{waitMessage},
		},
		{
			name: "wait with the wrong timeout",
			code: header + "- launchApp\n- extendedWaitUntil:\n    visible: \"credimi-never\"\n" +
				"    optional: true\n    timeout: 20000\n" + waitTail,
			appIDs:      wallet,
			waitSeconds: 240,
			want:        []string{waitMessage},
		},
		{
			name: "wait with a string timeout",
			code: header + "- launchApp\n- extendedWaitUntil:\n    visible: \"credimi-never\"\n" +
				"    optional: true\n    timeout: \"240000\"\n" + waitTail,
			appIDs:      wallet,
			waitSeconds: 240,
			want:        []string{waitMessage},
		},
		{
			name: "wait with an underscored timeout",
			code: header + "- launchApp\n- extendedWaitUntil:\n    visible: \"credimi-never\"\n" +
				"    optional: true\n    timeout: 240_000\n" + waitTail,
			appIDs:      wallet,
			waitSeconds: 240,
			want:        []string{waitMessage},
		},
		{
			name: "wait with a hex timeout",
			code: header + "- launchApp\n- extendedWaitUntil:\n    visible: \"credimi-never\"\n" +
				"    optional: true\n    timeout: 0x3A980\n" + waitTail,
			appIDs:      wallet,
			waitSeconds: 240,
			want:        []string{waitMessage},
		},
		{
			name:        "assertion last on a wait action",
			code:        header + "- launchApp\n" + codeTestTail,
			appIDs:      wallet,
			waitSeconds: 240,
			want:        []string{waitMessage},
		},
		{
			name: "wait action without a screenshot",
			code: header + "- launchApp\n- extendedWaitUntil:\n    visible: \"credimi-never\"\n" +
				"    optional: true\n    timeout: 240000\n",
			appIDs:      wallet,
			waitSeconds: 240,
		},
		{
			name: "optional wait on a non-wait action",
			code: header + "- launchApp\n- extendedWaitUntil:\n    notVisible: \"credimi-never\"\n" +
				"    optional: true\n    timeout: 240000\n" + waitTail,
			appIDs: wallet,
			want:   []string{lastMessage},
		},
		{
			name:   "outcome asserted on the screen before the last interaction",
			code:   header + shareInformation + "- assertVisible: \"Share information\"\n" + waitTail,
			appIDs: wallet,
			want: []string{"the final assertion checks \"Share information\", which the action " +
				"already reached before its last interaction; assert the screen the outcome shows"},
		},
		{
			name: "outcome waited by id after reaching it in a runFlow",
			code: header + "- launchApp\n- runFlow:\n    commands:\n" +
				"      - assertVisible:\n          id: \"share\"\n      - tapOn: \"CONFIRM\"\n" +
				"- extendedWaitUntil:\n    visible:\n      id: \"share\"\n    timeout: 5000\n",
			appIDs: wallet,
			want:   []string{staleMessage("id: share")},
		},
		{
			name:   "outcome asserted on a new screen",
			code:   header + shareInformation + "- assertVisible: \"Presentation shared\"\n" + waitTail,
			appIDs: wallet,
		},
		{
			name:   "text and id of the same value are different selectors",
			code:   header + shareInformation + "- assertVisible:\n    id: \"Share information\"\n",
			appIDs: wallet,
		},
		{
			name: "wait outcome on the screen before the last interaction",
			code: header + "- launchApp\n- assertVisible: \"credimi-never\"\n- tapOn: \"CONFIRM\"\n" +
				"- extendedWaitUntil:\n    visible: \"credimi-never\"\n" +
				"    optional: true\n    timeout: 240000\n" + waitTail,
			appIDs:      wallet,
			waitSeconds: 240,
		},
		{
			name: "outcome seen only in a when condition",
			code: header + "- launchApp\n- runFlow:\n    when:\n      visible: \"Share information\"\n" +
				"    commands:\n      - tapOn: \"CONFIRM\"\n- assertVisible: \"Share information\"\n",
			appIDs: wallet,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ValidateCode(tc.code, CodeScope{
				AppIDs:      tc.appIDs,
				Parameters:  tc.parameters,
				StoreLinks:  tc.storeLinks,
				WaitSeconds: tc.waitSeconds,
			}))
		})
	}
}
