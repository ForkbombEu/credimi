# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Maestro output helpers: inspect_screen compaction and `maestro test` result lines.

The fixtures follow what Maestro CLI 2.x printed on a Pixel 7a on 27/10/2026.
"""

import asyncio
import json

import pytest

from credimi_action_author.activities import _with_heartbeats
from credimi_action_author.maestro import (
    compact_hierarchy,
    output_tail,
    parse_run_output,
    strip_viewer_notice,
)

NOTICE = "Important: Maestro Viewer is available at http://127.0.0.1:10000/. Open it."

HIERARCHY = {
    "ui_schema": {
        "platform": "android",
        "abbreviations": {
            "b": "bounds",
            "txt": "text",
            "rid": "resource-id",
            "a11y": "content-desc",
            "hint": "hintText",
            "cls": "class",
            "scroll": "scrollable",
            "c": "children",
        },
        "defaults": {
            "enabled": True,
            "clickable": False,
            "focused": False,
            "selected": False,
            "checked": False,
            "scrollable": False,
            "txt": "",
            "hint": "",
            "rid": "",
            "a11y": "",
            "cls": "",
        },
    },
    "elements": [
        {
            "enabled": False,
            "c": [
                {
                    "b": "[0,0][1080,2400]",
                    "cls": "android.widget.FrameLayout",
                    "c": [
                        {
                            "b": "[63,360][1017,446]",
                            "txt": "Welcome to your Wallet",
                            "rid": "eu.europa.ec.euidi:id/pin_screen_title",
                            "cls": "android.widget.TextView",
                        },
                        {"b": "[63,900][1017,1100]", "cls": "android.view.View"},
                        {
                            "b": "[63,900][1017,1100]",
                            "hint": "PIN",
                            "cls": "android.widget.EditText",
                            "clickable": True,
                            "focused": True,
                        },
                        {
                            "b": "[22,146][148,272]",
                            "a11y": "Menu",
                            "cls": "android.view.View",
                            "enabled": False,
                        },
                        {"b": "[0,318][1080,2064]", "scroll": True, "cls": "ScrollView"},
                    ],
                }
            ],
        }
    ],
}


def test_compaction_keeps_only_targetable_nodes_one_per_line():
    raw = f"{NOTICE}\n{json.dumps(HIERARCHY)}"
    assert compact_hierarchy(raw).splitlines() == [
        "[63,360][1017,446] id=eu.europa.ec.euidi:id/pin_screen_title"
        ' text="Welcome to your Wallet"',
        '[63,900][1017,1100] hint="PIN" clickable,focused',
        '[22,146][148,272] desc="Menu" disabled',
    ]


def test_compaction_is_capped():
    big = {
        "ui_schema": HIERARCHY["ui_schema"],
        "elements": [{"b": f"[0,{i}][1,1]", "txt": "x" * 50} for i in range(500)],
    }
    text = compact_hierarchy(json.dumps(big), limit=1000)
    assert len(text) < 1100
    assert text.endswith("[... truncated]")


def test_compaction_of_non_json_returns_the_text():
    assert compact_hierarchy(f"{NOTICE}\nno device") == "no device"
    assert strip_viewer_notice(f"{NOTICE}\nrest") == "rest"


PASSED = """\
WARNING: A restricted method in java.lang.System has been called
Running on 37131JEHN05321
 > Flow FCAF precondition - EUDIW onboarding
Launch app "eu.europa.ec.euidi"... COMPLETED
Run flow when "Welcome to your Wallet" is visible...
  Tap on point (11%,42%)... COMPLETED
  Input text ${PIN}... COMPLETED
Run flow when "Welcome to your Wallet" is visible... COMPLETED
Run flow when "Welcome back" is visible...
Run flow when "Welcome back" is visible... SKIPPED
Take screenshot fcaf-onboarding-complete... COMPLETED
"""

FAILED = """\
Running on 37131JEHN05321
 > Flow fail
Launch app "eu.europa.ec.euidi"... COMPLETED
Assert that "Nonexistent thing" is visible... FAILED

Assertion is false: "Nonexistent thing" is visible

==== Debug output (logs & screenshots) ====
"""


def test_parse_completed_and_skipped_lines():
    run = parse_run_output(PASSED)
    assert [(r.command, r.status, r.depth) for r in run.results] == [
        ('Launch app "eu.europa.ec.euidi"', "COMPLETED", 0),
        ("Tap on point (11%,42%)", "COMPLETED", 1),
        ("Input text ${PIN}", "COMPLETED", 1),
        ('Run flow when "Welcome to your Wallet" is visible', "COMPLETED", 0),
        ('Run flow when "Welcome back" is visible', "SKIPPED", 0),
        ("Take screenshot fcaf-onboarding-complete", "COMPLETED", 0),
    ]
    assert run.failed is None
    assert run.completed == 5


def test_parse_failed_line():
    run = parse_run_output(FAILED)
    assert run.failed is not None
    assert run.failed.command == 'Assert that "Nonexistent thing" is visible'
    assert run.completed == 1


def test_output_tail_drops_jvm_noise_and_blank_lines():
    assert output_tail(PASSED, 2).splitlines() == [
        'Run flow when "Welcome back" is visible... SKIPPED',
        "Take screenshot fcaf-onboarding-complete... COMPLETED",
    ]
    assert "WARNING" not in output_tail(PASSED, 100)


async def test_cancelling_replay_wait_cancels_device_operation():
    started = asyncio.Event()
    stopped = asyncio.Event()

    async def device_operation():
        started.set()
        try:
            await asyncio.Event().wait()
        finally:
            stopped.set()

    task = asyncio.create_task(_with_heartbeats(device_operation()))
    await started.wait()
    task.cancel()
    with pytest.raises(asyncio.CancelledError):
        await task
    assert stopped.is_set()
