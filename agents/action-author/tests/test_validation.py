# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""validate_action: each rule, accepted and refused."""

import pytest

from credimi_action_author.fake_model import onboarding_action
from credimi_action_author.validation import validate_action

APP = "eu.europa.ec.euidi"
PARAMS = {"PIN": "123456"}

HEADER = f"appId: {APP}\nname: Onboarding\ntags:\n  - onboarding\n---\n"
TAIL = """\
- extendedWaitUntil:
    visible:
      id: "eu.europa.ec.euidi:id/dashboard_screen_bottom_navigation_item_home"
    timeout: 30000
- takeScreenshot: done
"""


def action(commands: str, header: str = HEADER, tail: str = TAIL) -> str:
    return header + commands + tail


GOOD = action('- launchApp\n- tapOn:\n    id: "pin"\n- inputText: "123456"\n')


def problems(text: str, params: dict[str, str] = PARAMS) -> list[str]:
    return validate_action(text, APP, "onboarding", params)


def test_a_valid_action_has_no_problems():
    assert problems(GOOD) == []
    assert problems(onboarding_action(APP, PARAMS["PIN"])) == []


def test_nested_inline_runflow_retry_and_repeat_are_valid():
    text = action(
        "- launchApp\n"
        "- runFlow:\n"
        "    when:\n"
        '      visible: "Welcome to your Wallet"\n'
        "    commands:\n"
        '      - inputText: "123456"\n'
        "      - retry:\n"
        "          maxRetries: 2\n"
        "          commands:\n"
        "            - hideKeyboard\n"
        "- repeat:\n"
        "    times: 2\n"
        "    commands:\n"
        "      - back\n"
    )
    assert problems(text) == []


# Rule 1: two documents.
@pytest.mark.parametrize(
    ("text", "expected"),
    [
        ("- launchApp\n", "expected two YAML documents"),
        (GOOD + "---\n- back\n", "expected two YAML documents"),
        ("appId: [unclosed\n---\n- back\n", "invalid YAML"),
        (f"- appId: {APP}\n---\n- launchApp\n", "first document must be the header"),
        (f"appId: {APP}\n---\nlaunchApp: true\n", "second document must be a non-empty list"),
    ],
)
def test_structure(text, expected):
    assert any(expected in p for p in problems(text)), problems(text)


# Rule 2: header.
def test_wrong_app_id_is_refused():
    text = GOOD.replace(f"appId: {APP}", "appId: com.example.other")
    assert any("header appId must be 'eu.europa.ec.euidi'" in p for p in problems(text))


def test_unknown_header_keys_are_refused():
    text = GOOD.replace("name: Onboarding", "name: Onboarding\nonFlowStart:\n  - back")
    assert any("header keys not allowed: ['onFlowStart']" in p for p in problems(text))


# Rule 3: command allowlist, nested commands walked.
def test_commands_outside_the_allowlist_are_refused_at_any_depth():
    text = action(
        '- launchApp\n- inputText: "123456"\n'
        "- runFlow:\n    commands:\n      - runScript: evil.js\n      - evalScript: ${1}\n"
    )
    found = problems(text)
    assert any("commands[2].runFlow.commands[0]: command `runScript`" in p for p in found)
    assert any("command `evalScript`" in p for p in found)


@pytest.mark.parametrize(
    "nested",
    [
        "- runFlow: other.yaml\n",
        "- runFlow:\n    file: other.yaml\n",
        "- retry:\n    file: other.yaml\n",
        "- repeat:\n    times: 2\n",
    ],
)
def test_runflow_retry_and_repeat_need_inline_commands(nested):
    found = problems(action('- launchApp\n- inputText: "123456"\n' + nested))
    assert any("inline `commands`" in p or "not allowed" in p for p in found), found


@pytest.mark.parametrize(
    "link",
    [
        "market://details?id=eu.europa.ec.euidi",
        "https://play.google.com/store/apps/details?id=x",
        "itms-apps://apps.apple.com/app/id1",
    ],
)
def test_openlink_to_an_app_store_is_refused(link):
    found = problems(action(f'- launchApp\n- inputText: "123456"\n- openLink: "{link}"\n'))
    assert any("openLink must not open an app store" in p for p in found), found


def test_openlink_to_a_web_page_is_allowed():
    text = action('- launchApp\n- inputText: "123456"\n- openLink:\n    link: https://x.eu/a\n')
    assert problems(text) == []


def test_launching_another_app_is_refused():
    found = problems(
        action('- launchApp:\n    appId: com.android.settings\n- inputText: "123456"\n')
    )
    assert any("launchApp targets 'com.android.settings'" in p for p in found), found


# Rule 4: a screenshot.
def test_missing_screenshot_is_refused():
    text = GOOD.replace("- takeScreenshot: done\n", "")
    assert any("takeScreenshot" in p for p in problems(text))


# Rule 5: self-contained literal inputs.
@pytest.mark.parametrize("pin", ["123456", "001234"])
def test_requested_pin_is_embedded_without_losing_leading_zeros(pin):
    candidate = action(f'- launchApp\n- inputText: "{pin}"\n')
    assert problems(candidate, params={"PIN": pin}) == []


@pytest.mark.parametrize("value", ['"${PIN}"', '"654321"', "123456"])
def test_external_wrong_or_unquoted_pin_is_refused(value):
    assert any("literal" in problem for problem in problems(action(f"- inputText: {value}\n")))


def test_external_pin_reference_is_refused_even_with_a_literal_elsewhere():
    candidate = action(
        '- inputText: "123456"\n- runFlow:\n    commands:\n      - inputText: "${PIN}"\n'
    )
    assert any("external parameter" in problem for problem in problems(candidate))


# Rule 6: contract end.
def test_the_last_command_must_assert_the_outcome():
    text = action('- launchApp\n- inputText: "123456"\n', tail="- back\n- takeScreenshot: x\n")
    assert any("must assert the outcome" in p for p in problems(text))


def test_extended_wait_without_visible_is_not_an_outcome_assertion():
    tail = "- extendedWaitUntil:\n    notVisible: Loading\n- takeScreenshot: x\n"
    text = action('- launchApp\n- inputText: "123456"\n', tail=tail)
    assert any("must assert the outcome" in p for p in problems(text))


def test_assert_visible_ends_an_action():
    tail = '- assertVisible: "Home"\n- takeScreenshot: a\n- takeScreenshot: b\n'
    assert problems(action('- launchApp\n- inputText: "123456"\n', tail=tail)) == []


@pytest.mark.parametrize(
    "mention",
    [
        "# PIN: 123456\n",
        'name: "Onboarding 123456"\n',
        'tags: ["123456"]\n',
    ],
)
def test_comments_and_display_metadata_do_not_satisfy_required_parameters(mention):
    header = f"appId: {APP}\n{mention}---\n"
    candidate = action("- launchApp\n", header=header)
    assert any("PIN" in problem and "literal" in problem for problem in problems(candidate))


def test_pin_only_in_flow_env_is_not_a_literal_input():
    header = f'appId: {APP}\nenv:\n  TEST_PIN: "123456"\n---\n'
    candidate = action(
        '- launchApp\n- runFlow:\n    commands:\n      - inputText: "${TEST_PIN}"\n',
        header=header,
    )
    assert any("literal" in problem for problem in problems(candidate))


@pytest.mark.parametrize("key", ["1", "null", "true"])
def test_nonstring_header_keys_are_rejected_without_crashing(key):
    candidate = GOOD.replace("name: Onboarding", f"{key}: invalid\nunknown: invalid")
    found = problems(candidate)
    assert any("header keys" in problem for problem in found)
    assert any("unknown" in problem for problem in found)
