// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/walletroles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		"extendedWaitUntil with visible: <text>, optional: true and timeout: 240000"
	waitTail := "- takeScreenshot: done\n"
	staleMessage := func(selector string) string {
		return "the final assertion checks \"" + selector + "\", which the action already " +
			"reached before its last interaction; assert the screen the outcome shows"
	}
	matchAllMessage := func(value string) string {
		return "the final assertion checks \"" + value + "\", which matches any screen; " +
			"assert text the outcome shows"
	}
	gboard := "com.google.android.inputmethod.latin:id/key_pos_number_line2_1"
	foreignIDMessage := func(value, pkg string) string {
		return "selector id \"" + value + "\" belongs to another app (" + pkg + "); target the " +
			"wallet's own elements, or tap the on-screen keyboard's keys by their text (e.g. tapOn: \"4\")"
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
			name: "keyboard id of another app in a runFlow when",
			code: header + "- launchApp\n- runFlow:\n    when:\n      visible: \"PIN\"\n    commands:\n" +
				"      - tapOn:\n          id: \"" + gboard + "\"\n" + codeTestTail,
			appIDs: wallet,
			want:   []string{foreignIDMessage(gboard, "com.google.android.inputmethod.latin")},
		},
		{
			name: "ids of one other app give one problem",
			code: header + "- launchApp\n- tapOn:\n    id: \"" + gboard + "\"\n- retry:\n" +
				"    maxRetries: 2\n    commands:\n      - tapOn:\n          text: \"OK\"\n" +
				"          below:\n            id: \"com.google.android.inputmethod.latin:id/key_pos_0\"\n" +
				codeTestTail,
			appIDs: wallet,
			want:   []string{foreignIDMessage(gboard, "com.google.android.inputmethod.latin")},
		},
		{
			name: "android, unprefixed and wallet ids",
			code: header + "- launchApp\n- tapOn:\n    id: \"android:id/button1\"\n- tapOn:\n    id: \"pin\"\n" +
				"- tapOn:\n    id: \"com.example.wallet:id/x\"\n" + codeTestTail,
			appIDs: wallet,
		},
		{
			name: "Play Store id on install-app",
			code: header + "- launchApp\n- tapOn:\n    id: \"com.android.vending:id/install\"\n" +
				codeTestTail,
			appIDs: installApp,
		},
		{
			name: "Play Store id outside install-app",
			code: header + "- launchApp\n- tapOn:\n    id: \"com.android.vending:id/install\"\n" +
				codeTestTail,
			appIDs: wallet,
			want: []string{
				foreignIDMessage("com.android.vending:id/install", "com.android.vending"),
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
			name: "final assertion that matches any screen",
			code: header + "- launchApp\n- tapOn: \"CONFIRM\"\n- extendedWaitUntil:\n" +
				"    visible: \".*\"\n    timeout: 5000\n" + waitTail,
			appIDs: wallet,
			want:   []string{matchAllMessage(".*")},
		},
		{
			name: "match-all assertion already reached reports only match-all",
			code: header + "- launchApp\n- assertVisible: \".*\"\n- tapOn: \"CONFIRM\"\n" +
				"- assertVisible: \".*\"\n",
			appIDs: wallet,
			want:   []string{matchAllMessage(".*")},
		},
		{
			name:   "id selector matching any value is not a text match-all",
			code:   header + "- launchApp\n- tapOn: \"CONFIRM\"\n- assertVisible:\n    id: \".*\"\n",
			appIDs: wallet,
		},
		{
			name: "wait action ending on a match-all wait",
			code: header + "- launchApp\n- extendedWaitUntil:\n    visible: \".*\"\n" +
				"    optional: true\n    timeout: 240000\n" + waitTail,
			appIDs:      wallet,
			waitSeconds: 240,
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

func TestValidateCodeMatchAll(t *testing.T) {
	wallet := AllowedAppIDs(codeTestWallet, "onboarding", PlatformAndroid)
	matchAll := []string{"", ".*", ".+", "(.*)", "^.*$", ".*|.*", " .* "}
	for _, value := range matchAll {
		t.Run("match-all "+strconv.Quote(value), func(t *testing.T) {
			code := "appId: com.example.wallet\n---\n- launchApp\n- tapOn: \"CONFIRM\"\n" +
				"- assertVisible: " + strconv.Quote(value) + "\n"
			assert.Equal(t, []string{"the final assertion checks \"" + value +
				"\", which matches any screen; assert text the outcome shows"},
				ValidateCode(code, CodeScope{AppIDs: wallet}))
		})
	}
	notMatchAll := []string{"Credentials|Wallet|Home|Add.*", "Credimi Demo PID.*", ".*Welcome.*"}
	for _, value := range notMatchAll {
		t.Run("not match-all "+strconv.Quote(value), func(t *testing.T) {
			code := "appId: com.example.wallet\n---\n- launchApp\n- tapOn: \"CONFIRM\"\n" +
				"- assertVisible: " + strconv.Quote(value) + "\n"
			assert.Empty(t, ValidateCode(code, CodeScope{AppIDs: wallet}))
		})
	}
}

// fcafTestPackages are the reference wallet and browser packages.
var fcafTestPackages = map[walletroles.App]string{
	walletroles.AppWallet:  "eu.europa.ec.euidi",
	walletroles.AppBrowser: "com.android.chrome",
}

// fcafRoleCodeScope is the code scope of an FCAF role of the reference wallet.
func fcafRoleCodeScope(role walletroles.Role) CodeScope {
	return CodeScope{
		AppIDs:      FCAFAppIDs(role, fcafTestPackages),
		Parameters:  role.Parameters,
		WaitSeconds: role.WaitSeconds,
		FCAF:        &FCAFCodeScope{Role: role, Packages: fcafTestPackages},
	}
}

func TestFCAFAppIDs(t *testing.T) {
	assert.Equal(t, []string{"eu.europa.ec.euidi"},
		FCAFAppIDs(walletroles.Role{HeaderApp: walletroles.AppWallet}, fcafTestPackages))
	assert.Equal(t, []string{"eu.europa.ec.euidi", "com.android.chrome"},
		FCAFAppIDs(walletroles.Role{
			HeaderApp:  walletroles.AppWallet,
			LaunchApps: []walletroles.App{walletroles.AppBrowser},
		}, fcafTestPackages))
	assert.Equal(t, []string{"eu.europa.ec.euidi", "com.android.chrome"},
		FCAFAppIDs(walletroles.Role{HeaderApp: walletroles.AppBrowser}, fcafTestPackages))
	// A wallet without a browser package does not allow an empty app id.
	assert.Equal(t, []string{"eu.europa.ec.euidi"},
		FCAFAppIDs(walletroles.Role{HeaderApp: walletroles.AppBrowser},
			map[walletroles.App]string{walletroles.AppWallet: "eu.europa.ec.euidi"}))
}

// TestValidateCodeAcceptsFCAFReferences checks every relying-party reference
// flow: only decorative screenshot names longer than their role's budget, and
// the problems listed below, are refused.
func TestValidateCodeAcceptsFCAFReferences(t *testing.T) {
	tolerated := map[string][]string{
		"fcaf-expect-credential-rejected":     {"credential-rejected"},
		"fcaf-engagement-haip-vp":             {"fcaf-engagement-haip-vp-request"},
		"fcaf-submit-request-object-by-value": {"fcaf-request-object-by-value-accepted"},
		"fcaf-present-all-credentials": {
			"fcaf-117-all-credentials-required-bundle",
			"fcaf-117-all-credentials-shared",
		},
		"fcaf-present-allowed-claim-path-components": {
			"fcaf-118-allowed-claim-path-components",
			"fcaf-118-allowed-claim-path-components-shared",
		},
		"fcaf-present-two-credentials": {
			"fcaf-main-interaction-039-shared",
			"fcaf-main-interaction-039-two-credentials",
		},
		"fcaf-present-same-credential-multiple-queries": {
			"fcaf-087-same-credential-multiple-queries",
		},
		"fcaf-user-authentication-failed": {
			"fcaf-149-consent-screen-before-authentication",
			"fcaf-149-user-authentication-failed",
		},
		"fcaf-exercise-http-error-redirect-response": {"http-error-redirect-wallet-error"},
	}
	// The reference wallet offers no control that deselects a claim (its row
	// only expands), so its flow cannot prove the state change.
	toleratedProblems := map[string][]string{
		"fcaf-present-unchecked-claim": {fcafStateChangeProblem("address")},
	}
	scope := fcafTestScope(t)
	for _, role := range scope.Contract.Roles {
		t.Run(role.Name, func(t *testing.T) {
			want := append([]string{}, toleratedProblems[role.Name]...)
			for _, name := range tolerated[role.Name] {
				want = append(want, fmt.Sprintf("screenshot %s is longer than %d characters, "+
					"the room the FCAF pipeline step ids leave", name, role.DiagnosticNameBudget))
			}
			problems := ValidateCode(role.ReferenceCode, fcafRoleCodeScope(role))
			assert.ElementsMatch(t, want, problems)
		})
	}
}

func TestValidateCodeFCAF(t *testing.T) {
	scope := fcafTestScope(t)
	// Every role that launches the wallet starts with launchApp; one that launches
	// nothing starts without it.
	const header = "appId: eu.europa.ec.euidi\n---\n- launchApp:\n    stopApp: true\n"
	const unlaunched = "appId: eu.europa.ec.euidi\n---\n"
	cases := []struct {
		name    string
		role    string
		code    string
		want    []string
		missing bool // want lists some of the problems only
		// edit changes the contract role; no contract screenshot is unconditional yet.
		edit func(*walletroles.Role)
	}{
		{
			name: "no final assertion needed",
			role: "fcaf-observe-request",
			code: header + "- openLink: ${deeplink}\n- takeScreenshot: observed\n- tapOn: \"Close\"\n",
		},
		{
			name: "reset clears the app state",
			role: "fcaf-reset-and-onboard",
			code: header + "- launchApp:\n    clearState: true\n- assertVisible: \"Home\"\n",
		},
		{
			name: "deliberate failures and stopApp",
			role: "fcaf-expect-request-rejected",
			code: header + "- stopApp\n- stopApp: eu.europa.ec.euidi\n- openLink: ${deeplink}\n" +
				"- runFlow:\n    when:\n      visible: \"Share\"\n    commands:\n" +
				"      - assertTrue:\n          condition: \"false\"\n          label: \"consent shown\"\n" +
				"      - assertTrue: false\n" +
				"- assertVisible: \"Home\"\n- takeScreenshot: request-refused\n",
		},
		{
			name: "missing contract screenshot",
			role: "fcaf-engagement-haip-vp",
			code: unlaunched + "- openLink: ${deeplink}\n- assertVisible: \"Requested data\"\n" +
				"- takeScreenshot: fcaf-engagement-haip-vp-invoked\n- tapOn: \"Share\"\n" +
				"- assertVisible: \"shared\"\n",
			want: []string{"the FCAF role fcaf-engagement-haip-vp must store the screenshot " +
				"fcaf-engagement-haip-vp-shared (gated: the wallet's confirmation that the " +
				"presentation was shared)"},
		},
		{
			name: "gated screenshot before any check",
			role: "fcaf-engagement-haip-vp",
			code: unlaunched + "- openLink: ${deeplink}\n" +
				"- takeScreenshot: fcaf-engagement-haip-vp-invoked\n- tapOn: \"Share\"\n" +
				"- assertVisible: \"shared\"\n- takeScreenshot: fcaf-engagement-haip-vp-shared\n",
			want: []string{"screenshot fcaf-engagement-haip-vp-invoked must be taken after a " +
				"check that proves the consent screen offering Share for this request: a " +
				"non-optional assertion before it, outside any runFlow when"},
		},
		{
			name: "while-screen screenshot at top level",
			role: "fcaf-expect-no-matching-document",
			code: header + "- openLink: ${deeplink}\n- takeScreenshot: no-match-notice\n",
			want: []string{"screenshot no-match-notice must be taken inside runFlow when: " +
				"visible, while the wallet's notice that it holds no credential matching the " +
				"request is shown"},
		},
		{
			name: "unconditional screenshot after a check",
			role: "fcaf-exercise-wallet-error",
			code: header + "- openLink: ${deeplink}\n- assertVisible: \"Error\"\n" +
				"- takeScreenshot: wallet-error\n",
			edit: func(role *walletroles.Role) {
				role.Screenshots[0].When = walletroles.ScreenshotUnconditional
			},
			want: []string{"screenshot wallet-error must be taken at top level, before any " +
				"non-optional check"},
		},
		{
			name: "silent role with a screenshot",
			role: "fcaf-issue-pid",
			code: header + "- openLink: ${deeplink}\n- assertVisible: \"Added\"\n- takeScreenshot: done\n",
			want: []string{"the FCAF role fcaf-issue-pid takes no screenshot"},
		},
		{
			name: "no screenshot",
			role: "fcaf-observe-request",
			code: header + "- openLink: ${deeplink}\n",
			want: []string{"the FCAF role fcaf-observe-request must store at least one " +
				"screenshot of the screen it reached"},
		},
		{
			name: "diagnostic name over budget",
			role: "fcaf-expect-request-rejected",
			code: header + "- openLink: ${deeplink}\n- takeScreenshot: launched-the-request\n" +
				"- assertVisible: \"Home\"\n- takeScreenshot: request-refused\n",
			want: []string{"screenshot launched-the-request is longer than 12 characters, " +
				"the room the FCAF pipeline step ids leave"},
		},
		{
			name: "wait role must wait",
			role: "fcaf-wait-past-status-list-expiry",
			code: "appId: eu.europa.ec.euidi\n---\n- assertVisible: \"Home\"\n",
			want: []string{"the last command other than takeScreenshot must wait 240 s: " +
				"extendedWaitUntil with visible: <text>, optional: true and timeout: 240000"},
		},
		{
			name: "launch of the wallet by package",
			role: "fcaf-expect-request-rejected",
			code: "appId: eu.europa.ec.euidi\n---\n- stopApp\n- launchApp: eu.europa.ec.euidi\n" +
				"- openLink: ${deeplink}\n- assertVisible: \"Home\"\n- takeScreenshot: request-refused\n",
		},
		{
			name: "no launch of the wallet",
			role: "fcaf-expect-request-rejected",
			code: "appId: eu.europa.ec.euidi\n---\n- openLink: ${deeplink}\n" +
				"- runFlow:\n    when:\n      visible: \"Locked\"\n    commands:\n      - launchApp\n" +
				"- assertVisible: \"Home\"\n- takeScreenshot: request-refused\n",
			want: []string{
				"the role launches the wallet: start the action with launchApp of eu.europa.ec.euidi",
				fcafStartProblem("fcaf-expect-request-rejected"),
			},
		},
		{
			name: "launch of another app only",
			role: "fcaf-expect-request-rejected",
			code: "appId: eu.europa.ec.euidi\n---\n- launchApp:\n    appId: com.android.chrome\n" +
				"- openLink: ${deeplink}\n- assertVisible: \"Home\"\n- takeScreenshot: request-refused\n",
			want: []string{
				"the role launches the wallet: start the action with launchApp of eu.europa.ec.euidi",
				"commands[0]: launchApp targets 'com.android.chrome', not the app under test",
			},
		},
		{
			name: "launch of the wallet in a role that launches nothing",
			role: "fcaf-engagement-haip-vp",
			code: header + "- openLink: ${deeplink}\n- assertVisible: \"Requested data\"\n" +
				"- takeScreenshot: fcaf-engagement-haip-vp-invoked\n- tapOn: \"Share\"\n" +
				"- assertVisible: \"shared\"\n- takeScreenshot: fcaf-engagement-haip-vp-shared\n",
			want: []string{
				"the role does not launch the wallet: start like the reference, without launchApp",
			},
		},
		{
			name: "clearState outside the reset role",
			role: "fcaf-expect-request-rejected",
			code: header + "- launchApp:\n    clearState: true\n- openLink: ${deeplink}\n" +
				"- assertVisible: \"Home\"\n- takeScreenshot: request-refused\n",
			want: []string{"commands[1]: only the FCAF reset role clears the app state"},
		},
		{
			name: "clearState quoted true outside the reset role",
			role: "fcaf-expect-request-rejected",
			code: header + "- launchApp:\n    clearState: \"true\"\n- openLink: ${deeplink}\n" +
				"- assertVisible: \"Home\"\n- takeScreenshot: request-refused\n",
			want: []string{"commands[1]: only the FCAF reset role clears the app state"},
		},
		{
			name: "clearState yes outside the reset role",
			role: "fcaf-expect-request-rejected",
			code: header + "- launchApp:\n    clearState: yes\n- openLink: ${deeplink}\n" +
				"- assertVisible: \"Home\"\n- takeScreenshot: request-refused\n",
			want: []string{"commands[1]: only the FCAF reset role clears the app state"},
		},
		{
			name: "clearState on outside the reset role",
			role: "fcaf-expect-request-rejected",
			code: header + "- launchApp:\n    clearState: on\n- openLink: ${deeplink}\n" +
				"- assertVisible: \"Home\"\n- takeScreenshot: request-refused\n",
			want: []string{"commands[1]: only the FCAF reset role clears the app state"},
		},
		{
			name: "clearState false outside the reset role",
			role: "fcaf-expect-request-rejected",
			code: header + "- launchApp:\n    clearState: false\n- openLink: ${deeplink}\n" +
				"- assertVisible: \"Home\"\n- takeScreenshot: request-refused\n",
		},
		{
			name: "assertTrue on an expression",
			role: "fcaf-expect-request-rejected",
			code: header + "- openLink: ${deeplink}\n- assertTrue: \"${x}\"\n" +
				"- assertVisible: \"Home\"\n- takeScreenshot: request-refused\n",
			want: []string{"commands[2]: assertTrue may only fail the flow on purpose: " +
				"assertTrue: {condition: \"false\"}"},
			missing: true,
		},
		{
			name: "assertTrue with another key",
			role: "fcaf-expect-request-rejected",
			code: header + "- openLink: ${deeplink}\n" +
				"- assertTrue:\n    condition: \"false\"\n    optional: true\n" +
				"- assertVisible: \"Home\"\n- takeScreenshot: request-refused\n",
			want: []string{"commands[2]: assertTrue may only fail the flow on purpose: " +
				"assertTrue: {condition: \"false\"}"},
		},
		{
			name: "stopApp of another app",
			role: "fcaf-expect-request-rejected",
			code: header + "- stopApp: com.android.chrome\n- openLink: ${deeplink}\n" +
				"- assertVisible: \"Home\"\n- takeScreenshot: request-refused\n",
			want: []string{
				"commands[1]: stopApp targets 'com.android.chrome', not the app under test",
			},
		},
		{
			name: "browser header for a wallet role",
			role: "fcaf-engagement-haip-vp",
			code: "appId: com.android.chrome\n---\n- openLink: ${deeplink}\n" +
				"- assertVisible: \"Share\"\n- takeScreenshot: fcaf-engagement-haip-vp-invoked\n" +
				"- assertVisible: \"shared\"\n- takeScreenshot: fcaf-engagement-haip-vp-shared\n",
			want: []string{"header appId must be 'eu.europa.ec.euidi' for the FCAF role " +
				"fcaf-engagement-haip-vp, found 'com.android.chrome'"},
			missing: true,
		},
		{
			name: "when detects a screen by an enabled control",
			role: "fcaf-expect-request-rejected",
			code: header + "- openLink: ${deeplink}\n" +
				"- runFlow:\n    when:\n      visible:\n        text: \"Share\"\n        enabled: true\n" +
				"    commands:\n      - assertTrue: false\n" +
				"- runFlow:\n    when:\n      notVisible:\n        text: \"Share\"\n        selected: false\n" +
				"    commands:\n      - assertVisible: \"Home\"\n" +
				"- assertVisible: \"Home\"\n- takeScreenshot: request-refused\n",
			want: []string{fcafStateConditionProblem},
		},
		{
			name: "nested when checks a control's state",
			role: "fcaf-expect-request-rejected",
			code: header + "- openLink: ${deeplink}\n" +
				"- retry:\n    maxRetries: 2\n    commands:\n" +
				"      - runFlow:\n          when:\n            visible: \"Home\"\n          commands:\n" +
				"            - runFlow:\n                when:\n                  notVisible:\n" +
				"                    id: \"eu.europa.ec.euidi:id/agree\"\n                    checked: true\n" +
				"                commands:\n                  - tapOn: \"Close\"\n" +
				"- assertVisible: \"Home\"\n- takeScreenshot: request-refused\n",
			want: []string{fcafStateConditionProblem},
		},
		{
			name: "control state outside when",
			role: "fcaf-expect-request-rejected",
			code: header + "- openLink: ${deeplink}\n- assertVisible: \"Requested data\"\n" +
				"- assertVisible:\n    text: \"Share\"\n    enabled: true\n" +
				"- tapOn:\n    text: \"Share\"\n    focused: true\n" +
				"- runFlow:\n    when:\n      visible: \"Share\"\n    commands:\n" +
				"      - tapOn:\n          text: \"Share\"\n          enabled: true\n" +
				"- assertVisible: \"Home\"\n- takeScreenshot: request-refused\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			role, ok := scope.Contract.Role(tc.role)
			require.True(t, ok)
			if tc.edit != nil {
				role.Screenshots = append([]walletroles.Screenshot{}, role.Screenshots...)
				tc.edit(&role)
			}
			problems := ValidateCode(tc.code, fcafRoleCodeScope(role))
			if tc.missing {
				assert.Subset(t, problems, tc.want)
				return
			}
			if tc.want == nil {
				tc.want = []string{}
			}
			assert.Equal(t, tc.want, append([]string{}, problems...))
		})
	}
}

// fcafProblems validates code as the action of the named reference role.
func fcafProblems(t *testing.T, roleName, code string) []string {
	t.Helper()
	role, ok := fcafTestScope(t).Contract.Role(roleName)
	require.True(t, ok)
	return ValidateCode(code, fcafRoleCodeScope(role))
}

func TestValidateCodeFCAFTapAnchors(t *testing.T) {
	const rejected = "appId: eu.europa.ec.euidi\n---\n- launchApp:\n    stopApp: true\n" +
		"- openLink: ${deeplink}\n"
	const rejectedEnd = "- assertVisible: \"Home\"\n- takeScreenshot: request-refused\n"
	share := fcafTapAnchorProblem("Share")
	cases := []struct {
		name    string
		role    string
		code    string
		refused bool
	}{
		{
			name: "when visible Share with a notVisible notice",
			role: "fcaf-expect-request-rejected",
			code: rejected + "- runFlow:\n    when:\n      visible: \"Share\"\n" +
				"      notVisible: \"The requested document is not available.*\"\n" +
				"    commands:\n      - tapOn: \"Share\"\n      - assertTrue: false\n" + rejectedEnd,
			refused: true,
		},
		{
			name: "when visible lists Share and the branch goes on after the tap",
			role: "fcaf-expect-request-rejected",
			code: rejected + "- runFlow:\n    when:\n      visible: \"Requested data|Share\"\n" +
				"    commands:\n      - tapOn:\n          text: \"Share\"\n          enabled: true\n" +
				"      - inputText: \"12345\"\n      - assertTrue: false\n" + rejectedEnd,
			refused: true,
		},
		{
			name: "assertVisible Share then tap Share",
			role: "fcaf-expect-request-rejected",
			code: rejected + "- assertVisible: \"Share\"\n- takeScreenshot: consent\n" +
				"- tapOn: \"Share\"\n" + rejectedEnd,
			refused: true,
		},
		{
			name: "strict wait listing Share then tap Share",
			role: "fcaf-engagement-haip-vp",
			code: "appId: eu.europa.ec.euidi\n---\n- openLink: ${deeplink}\n" +
				"- extendedWaitUntil:\n    visible: \"Share|Cancel\"\n    timeout: 30000\n" +
				"- tapOn: \"Share\"\n",
			refused: true,
		},
		{
			name: "scrollUntilVisible Share alone then tap Share",
			role: "fcaf-engagement-haip-vp",
			code: "appId: eu.europa.ec.euidi\n---\n- openLink: ${deeplink}\n" +
				"- scrollUntilVisible:\n    element: \"Share\"\n- tapOn: \"Share\"\n",
			refused: true,
		},
		{
			name: "Share tapped after asserting the requested data",
			role: "fcaf-engagement-haip-vp",
			code: "appId: eu.europa.ec.euidi\n---\n- openLink: ${deeplink}\n" +
				"- assertVisible: \"Requested data\"\n- scrollUntilVisible:\n    element: \"Share\"\n" +
				"- assertVisible: \"Share\"\n- tapOn: \"Share\"\n",
		},
		{
			name: "optional wait listing Share among other screens",
			role: "fcaf-engagement-haip-vp",
			code: "appId: eu.europa.ec.euidi\n---\n- openLink: ${deeplink}\n" +
				"- extendedWaitUntil:\n    visible: \"Requested data|Share|Oups! Something went wrong\"\n" +
				"    timeout: 30000\n    optional: true\n- tapOn: \"Share\"\n",
		},
		{
			name: "consent branch identified by the requested data",
			role: "fcaf-expect-request-rejected",
			code: rejected + "- runFlow:\n    when:\n      visible: \"Requested data\"\n" +
				"    commands:\n      - tapOn: \"Share\"\n      - inputText: \"12345\"\n" +
				"      - assertTrue: false\n" + rejectedEnd,
		},
		{
			name: "dismissal idiom",
			role: "fcaf-expect-request-rejected",
			code: rejected + "- runFlow:\n    when:\n      visible: \"Share\"\n" +
				"    commands:\n      - tapOn: \"Share\"\n" + rejectedEnd,
		},
		{
			name: "dismissal followed only by a wait and a screenshot",
			role: "fcaf-expect-request-rejected",
			code: rejected + "- runFlow:\n    when:\n      visible: \"Close|Done\"\n" +
				"    commands:\n      - tapOn: \"Close\"\n" +
				"      - extendedWaitUntil:\n          visible: \"Home\"\n          timeout: 3000\n" +
				"      - takeScreenshot: closed\n" + rejectedEnd,
		},
		{
			name: "dismissal followed by a PIN at the same level",
			role: "fcaf-expect-request-rejected",
			code: rejected + "- runFlow:\n    when:\n      visible: \"Share\"\n" +
				"    commands:\n      - tapOn: \"Share\"\n- takeScreenshot: after\n" +
				"- inputText: \"12345\"\n" + rejectedEnd,
			refused: true,
		},
		{
			name: "dismissal followed by a strict anchor, then an interaction",
			role: "fcaf-expect-request-rejected",
			code: rejected + "- runFlow:\n    when:\n      visible: \"Share\"\n" +
				"    commands:\n      - tapOn: \"Share\"\n- assertVisible: \"Home\"\n" +
				"- tapOn: \"Close\"\n" + rejectedEnd,
		},
		{
			name: "dismissal followed by a conditional PIN block",
			role: "fcaf-expect-request-rejected",
			code: rejected + "- runFlow:\n    when:\n      visible: \"Just once\"\n" +
				"    commands:\n      - tapOn: \"Just once\"\n" +
				"- runFlow:\n    when:\n      visible: \"Welcome back\"\n    commands:\n" +
				"      - inputText: \"12345\"\n" + rejectedEnd,
		},
		{
			name: "assertVisible Share then tap Share inside retry",
			role: "fcaf-expect-request-rejected",
			code: rejected + "- retry:\n    maxRetries: 2\n    commands:\n" +
				"      - assertVisible: \"Share\"\n      - tapOn: \"Share\"\n" + rejectedEnd,
			refused: true,
		},
		{
			name: "when visible Share with notVisible inside retry",
			role: "fcaf-expect-request-rejected",
			code: rejected + "- retry:\n    maxRetries: 2\n    commands:\n" +
				"      - runFlow:\n          when:\n            visible: \"Share\"\n" +
				"            notVisible: \"x\"\n          commands:\n" +
				"            - tapOn: \"Share\"\n" + rejectedEnd,
			refused: true,
		},
		{
			name: "when visible Share inside repeat, then a PIN in the body",
			role: "fcaf-expect-request-rejected",
			code: rejected + "- repeat:\n    times: 2\n    commands:\n" +
				"      - runFlow:\n          when:\n            visible: \"Share\"\n" +
				"          commands:\n            - tapOn: \"Share\"\n" +
				"      - inputText: \"12345\"\n" + rejectedEnd,
			refused: true,
		},
		{
			name: "role that never taps to share",
			role: "fcaf-user-denied-consent",
			code: "appId: eu.europa.ec.euidi\n---\n- launchApp:\n    stopApp: true\n" +
				"- openLink: ${deeplink}\n- assertVisible: \"Deny\"\n- tapOn: \"Deny\"\n" +
				"- takeScreenshot: denied\n",
		},
		{
			name: "rejection role that accepts an offer",
			role: "fcaf-expect-credential-rejected",
			code: "appId: eu.europa.ec.euidi\n---\n- launchApp:\n    stopApp: true\n" +
				"- openLink: ${deeplink}\n- extendedWaitUntil:\n    visible: \"Accept\"\n" +
				"    timeout: 100000\n- tapOn: \"Accept\"\n- takeScreenshot: offer\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			problems := fcafProblems(t, tc.role, tc.code)
			if tc.refused {
				assert.Contains(t, problems, share)
				return
			}
			for _, problem := range problems {
				assert.NotContains(t, problem, "the screen before tapping")
			}
		})
	}
	assert.Equal(t, "the screen before tapping \"Share\" is identified only by \"Share\" itself; "+
		"assert what the screen shows (on the consent screen, the requested data), as the "+
		"reference flow does", share)
}

func TestValidateCodeFCAFStart(t *testing.T) {
	const body = "- openLink: ${deeplink}\n- assertVisible: \"Home\"\n" +
		"- takeScreenshot: request-refused\n"
	const head = "appId: eu.europa.ec.euidi\n---\n"
	problem := fcafStartProblem("fcaf-expect-request-rejected")
	assert.Equal(t, "the reference flow of fcaf-expect-request-rejected starts by stopping the "+
		"app; start the action with stopApp, or launchApp without stopApp: false", problem)
	for _, tc := range []struct {
		name    string
		role    string
		code    string
		refused bool
	}{
		{"no stop first", "fcaf-expect-request-rejected", head + "- openLink: ${deeplink}\n" +
			"- launchApp\n- assertVisible: \"Home\"\n- takeScreenshot: request-refused\n", true},
		{"stopApp false", "fcaf-expect-request-rejected",
			head + "- launchApp:\n    stopApp: false\n" + body, true},
		{"plain launchApp", "fcaf-expect-request-rejected", head + "- launchApp\n" + body, false},
		{"launchApp of the wallet", "fcaf-expect-request-rejected",
			head + "- launchApp: eu.europa.ec.euidi\n" + body, false},
		{"launchApp without stopApp", "fcaf-expect-request-rejected",
			head + "- launchApp:\n    clearState: false\n" + body, false},
		{"stopApp quoted false", "fcaf-expect-request-rejected",
			head + "- launchApp:\n    stopApp: \"false\"\n" + body, false},
		{"launchApp with stopApp", "fcaf-expect-request-rejected",
			head + "- launchApp:\n    stopApp: true\n" + body, false},
		{"stopApp then launchApp", "fcaf-expect-request-rejected",
			head + "- stopApp\n- launchApp\n" + body, false},
		{"reference starting with stopApp", "fcaf-submit-and-follow-redirect",
			head + "- openLink: ${deeplink}\n- takeScreenshot: x\n", true},
		{"reference starting with openLink", "fcaf-engagement-haip-vp",
			head + "- openLink: ${deeplink}\n- takeScreenshot: x\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			role, ok := fcafTestScope(t).Contract.Role(tc.role)
			require.True(t, ok)
			problems := ValidateCode(tc.code, fcafRoleCodeScope(role))
			assert.Equal(
				t,
				tc.refused,
				slices.Contains(problems, fcafStartProblem(tc.role)),
				problems,
			)
		})
	}
}

func TestValidateCodeFCAFStateChange(t *testing.T) {
	const head = "appId: eu.europa.ec.euidi\n---\n- launchApp:\n    stopApp: true\n" +
		"- openLink: ${deeplink}\n- assertVisible: \"Requested data\"\n"
	const tail = "- tapOn: \"Share\"\n- assertVisible: \".*successfully shared.*\"\n" +
		"- takeScreenshot: shared\n"
	const checkedID = "- assertVisible:\n    id: \"eu.europa.ec.euidi:id/address_switch\"\n" +
		"    checked: %s\n"
	const tapID = "- tapOn:\n    id: \"eu.europa.ec.euidi:id/address_switch\"\n"
	problem := fcafStateChangeProblem("address")
	assert.Equal(t, "this role must prove the address control changes state: assert it "+
		"checked before the tap and unchecked after, on the same element (a row that only "+
		"expands has no such control)", problem)
	for _, tc := range []struct {
		name    string
		code    string
		refused bool
	}{
		{
			name: "checked before and unchecked after on the same id",
			code: head + fmt.Sprintf(checkedID, "true") + tapID + fmt.Sprintf(checkedID, "false") + tail,
		},
		{
			name: "checked before and unchecked after on the same text",
			code: head + "- extendedWaitUntil:\n    visible:\n      text: \"Address\"\n" +
				"      checked: true\n    timeout: 5000\n- tapOn: \"Address\"\n" +
				"- assertVisible:\n    text: \"Address\"\n    checked: false\n" + tail,
		},
		{
			name:    "unchecked after only",
			code:    head + tapID + fmt.Sprintf(checkedID, "false") + tail,
			refused: true,
		},
		{
			name:    "row that only expands",
			code:    head + "- tapOn: \"address\"\n- assertVisible: \"Street\"\n" + tail,
			refused: true,
		},
		{
			name: "checked and unchecked on different elements",
			code: head + "- assertVisible:\n    text: \"Address\"\n    checked: true\n" + tapID +
				fmt.Sprintf(checkedID, "false") + tail,
			refused: true,
		},
		{
			name: "no tap between the assertions",
			code: head + fmt.Sprintf(checkedID, "true") + fmt.Sprintf(checkedID, "false") +
				tapID + tail,
			refused: true,
		},
		{
			name: "optional unchecked wait",
			code: head + fmt.Sprintf(checkedID, "true") + tapID +
				"- extendedWaitUntil:\n    visible:\n      id: \"eu.europa.ec.euidi:id/address_switch\"\n" +
				"      checked: false\n    timeout: 5000\n    optional: true\n" + tail,
			refused: true,
		},
		{
			name: "proof after expanding the document card",
			code: head + "- tapOn: \"PID\"\n" +
				"- assertVisible:\n    text: \"Address\"\n    checked: true\n- tapOn: \"Address\"\n" +
				"- assertVisible:\n    text: \"Address\"\n    checked: false\n" + tail,
		},
		{
			name: "proof inside retry",
			code: head + fmt.Sprintf(checkedID, "true") + "- retry:\n    maxRetries: 2\n" +
				"    commands:\n      - tapOn:\n          id: \"eu.europa.ec.euidi:id/address_switch\"\n" +
				"      - assertVisible:\n          id: \"eu.europa.ec.euidi:id/address_switch\"\n" +
				"          checked: false\n" + tail,
		},
		{
			name: "proof inside a conditional runFlow",
			code: head + "- runFlow:\n    when:\n      visible:\n" +
				"        id: \"eu.europa.ec.euidi:id/address_switch\"\n    commands:\n" +
				"      - assertVisible:\n          id: \"eu.europa.ec.euidi:id/address_switch\"\n" +
				"          checked: true\n      - tapOn: \"Address\"\n" +
				"      - assertVisible:\n          id: \"eu.europa.ec.euidi:id/address_switch\"\n" +
				"          checked: false\n" + tail,
			refused: true,
		},
		{
			name: "switch found right of its label",
			code: head + "- assertVisible:\n    id: \"eu.europa.ec.euidi:id/switch\"\n" +
				"    rightOf: \"Address\"\n    checked: true\n- tapOn: \"Address\"\n" +
				"- assertVisible:\n    rightOf: \"Address\"\n    checked: false\n" +
				"    id: \"eu.europa.ec.euidi:id/switch\"\n" + tail,
		},
		{
			name: "switch right of another label after the tap",
			code: head + "- assertVisible:\n    id: \"eu.europa.ec.euidi:id/switch\"\n" +
				"    rightOf: \"Address\"\n    checked: true\n- tapOn: \"Address\"\n" +
				"- assertVisible:\n    id: \"eu.europa.ec.euidi:id/switch\"\n" +
				"    rightOf: \"Address country\"\n    checked: false\n" + tail,
			refused: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			problems := fcafProblems(t, "fcaf-present-unchecked-claim", tc.code)
			assert.Equal(t, tc.refused, slices.Contains(problems, problem), problems)
			if !tc.refused {
				for _, p := range problems {
					assert.NotContains(t, p, "the screen before tapping")
				}
			}
		})
	}
	// Another role proves nothing.
	assert.NotContains(t, fcafProblems(t, "fcaf-engagement-haip-vp",
		"appId: eu.europa.ec.euidi\n---\n- openLink: ${deeplink}\n- takeScreenshot: x\n"), problem)
}

func TestValidateCodeChatRefusesFCAFCommands(t *testing.T) {
	code := "appId: com.example.wallet\n---\n- stopApp\n- assertTrue: false\n" +
		"- assertVisible: \"Welcome\"\n"
	assert.Equal(t, []string{
		"commands[0]: command `stopApp` is not allowed",
		"commands[1]: command `assertTrue` is not allowed",
	}, ValidateCode(code, CodeScope{AppIDs: []string{codeTestWallet}}))
}
