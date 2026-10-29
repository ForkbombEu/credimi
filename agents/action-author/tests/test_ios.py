# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""IOSSimulatorDevice against a scripted `xcrun simctl`: build checks and clone-per-reset."""

import hashlib
import json
from pathlib import Path

import pytest

from credimi_action_author import ios
from credimi_action_author.contracts import PrepareBuildRequest, ResolvedBuild

BASE = "11111111-2222-3333-4444-555555555555"
BUNDLE = "eu.europa.ec.euidi-dev"
DIGEST = hashlib.sha256(b"zipped app").hexdigest()


@pytest.mark.parametrize(
    ("names", "root"),
    [
        (
            ["Payload/", "Payload/Wallet.app/Info.plist", "Payload/Wallet.app/Wallet"],
            "Payload/Wallet.app",
        ),
        (["Wallet.app/Info.plist", "__MACOSX/Wallet.app/._Info.plist"], "Wallet.app"),
    ],
)
def test_app_root_finds_the_single_bundle(names, root):
    assert str(ios.app_root(names)) == root


@pytest.mark.parametrize(
    ("names", "error"),
    [
        (["A.app/Info.plist", "B.app/Info.plist"], "exactly one .app bundle, found 2"),
        (["readme.txt"], "exactly one .app bundle, found 0"),
        (["../Wallet.app/Info.plist"], "unsafe path"),
        (["/Wallet.app/Info.plist"], "unsafe path"),
    ],
)
def test_app_root_rejects_ambiguous_or_unsafe_archives(names, error):
    with pytest.raises(ValueError, match=error):
        ios.app_root(names)


def plist_runner(values: dict[str, str]):
    async def run(cmd, timeout, cwd=None):
        key = cmd[2]
        return (0, values[key]) if key in values else (1, f"No value at that key path: {key}")

    return run


async def test_device_builds_are_refused(monkeypatch, tmp_path):
    monkeypatch.setattr(
        ios,
        "_run",
        plist_runner({"CFBundleIdentifier": BUNDLE, "CFBundleSupportedPlatforms": '["iPhoneOS"]'}),
    )
    with pytest.raises(ValueError, match="not a simulator build"):
        await ios.read_app_info(tmp_path)


async def test_simulator_app_info(monkeypatch, tmp_path):
    monkeypatch.setattr(
        ios,
        "_run",
        plist_runner(
            {
                "CFBundleIdentifier": BUNDLE,
                "CFBundleSupportedPlatforms": '["iPhoneSimulator"]',
                "CFBundleVersion": "42",
                "CFBundleShortVersionString": "1.2.0",
            }
        ),
    )
    assert await ios.read_app_info(tmp_path) == (BUNDLE, "42", "1.2.0")


class FakeSimctl:
    """Clones, boots, installs and deletes simulators in memory; records every command."""

    def __init__(self):
        self.devices = {BASE: {"name": "credimi", "state": "Shutdown", "apps": set()}}
        self.log: list[tuple[str, ...]] = []
        self.counter = 0
        # get_app_container answers "No such app" this many times after each boot.
        self.unready_lookups = 0
        self.pending: dict[str, int] = {}

    async def command(self, cmd, timeout=60):
        code, out = await self.run(cmd, timeout)
        if code:
            raise RuntimeError(f"{cmd[0]} failed: {out}")
        return out

    async def run(self, cmd, timeout, cwd=None):
        assert cmd[:2] == ["xcrun", "simctl"], cmd
        verb, args = cmd[2], cmd[3:]
        self.log.append((verb, *args))
        if verb == "list":
            devices = [
                {"udid": u, "name": d["name"], "state": d["state"]} for u, d in self.devices.items()
            ]
            return 0, json.dumps({"devices": {"iOS": devices}})
        if verb == "clone":
            source, name = args
            assert self.devices[source]["state"] == "Shutdown"
            self.counter += 1
            udid = f"{self.counter:08X}-0000-0000-0000-000000000000"
            self.devices[udid] = {
                "name": name,
                "state": "Shutdown",
                "apps": set(self.devices[source]["apps"]),
            }
            return 0, udid
        device = self.devices.get(args[0])
        if device is None:
            return 1, "Invalid device"
        if verb == "boot":
            device["state"] = "Booted"
            self.pending[args[0]] = self.unready_lookups
        elif verb == "shutdown":
            device["state"] = "Shutdown"
        elif verb == "delete":
            del self.devices[args[0]]
        elif verb == "install":
            device["apps"].add(BUNDLE)
        elif verb == "get_app_container":
            if self.pending.get(args[0], 0) > 0:
                self.pending[args[0]] -= 1
                return 1, "No such app"
            return (0, "/container") if args[1] in device["apps"] else (1, "No such app")
        return 0, ""


@pytest.fixture
def simulator(monkeypatch, tmp_path):
    simctl = FakeSimctl()
    monkeypatch.setattr(ios, "require_tools", lambda *tools: None)
    monkeypatch.setattr(ios, "APP_READY_INTERVAL", 0)
    monkeypatch.setattr(ios, "_run", simctl.run)
    monkeypatch.setattr(ios, "command", simctl.command)
    monkeypatch.setattr(ios, "close_agent_device", _no_session)

    async def download(req, destination):
        destination.write_bytes(b"zipped app")

    async def extract(archive, destination):
        app = destination / "Payload" / "Wallet.app"
        app.mkdir(parents=True)
        return app

    async def info(app: Path):
        return BUNDLE, "42", "1.2.0"

    monkeypatch.setattr(ios, "download_installer", download)
    monkeypatch.setattr(ios, "extract_app", extract)
    monkeypatch.setattr(ios, "read_app_info", info)
    return ios.IOSSimulatorDevice(BASE), simctl


async def _no_session(device_args):
    return ""


def ios_build(platform="ios"):
    return PrepareBuildRequest(
        build=ResolvedBuild(
            wallet_id="wallet",
            version_id="version",
            version_identifier="organization/wallet/version",
            owner_namespace="organization",
            platform=platform,
            installer_name="wallet.zip",
            installer_sha256=DIGEST,
            download_grant="test-only-capability",
        )
    )


async def test_prepare_installs_into_a_template_and_boots_a_clone(simulator):
    device, simctl = simulator
    installed = await device.prepare_build(ios_build())

    assert (installed.package_id, installed.version_code, installed.version_name) == (
        BUNDLE,
        "42",
        "1.2.0",
    )
    assert installed.installer_sha256 == DIGEST
    # The base simulator is cloned, never booted or installed into.
    assert simctl.devices[BASE] == {"name": "credimi", "state": "Shutdown", "apps": set()}
    template = next(u for u, d in simctl.devices.items() if d["name"].endswith("-template"))
    assert simctl.devices[template]["state"] == "Shutdown"
    assert simctl.devices[device.serial] == {
        "name": f"credimi-action-author-{BASE}-clone-1",
        "state": "Booted",
        "apps": {BUNDLE},
    }
    await device.verify_build(BUNDLE, DIGEST)


async def test_every_reset_replaces_the_clone(simulator):
    device, simctl = simulator
    await device.prepare_build(ios_build())
    first = device.serial

    assert (await device.clear_app(BUNDLE)).endswith(device.serial)

    assert device.serial != first
    assert first not in simctl.devices
    assert simctl.devices[device.serial]["state"] == "Booted"
    clones = [d["name"] for d in simctl.devices.values() if "-clone-" in d["name"]]
    assert clones == [f"credimi-action-author-{BASE}-clone-2"]


async def test_a_reset_waits_until_the_clone_lists_the_wallet(simulator):
    device, simctl = simulator
    await device.prepare_build(ios_build())
    simctl.unready_lookups = 2

    await device.clear_app(BUNDLE)

    lookups = [e for e in simctl.log if e[:2] == ("get_app_container", device.serial)]
    assert len(lookups) == 3
    await device.verify_build(BUNDLE, DIGEST)


async def test_a_clone_that_never_lists_the_wallet_fails_the_reset(simulator, monkeypatch):
    device, simctl = simulator
    await device.prepare_build(ios_build())
    monkeypatch.setattr(ios, "APP_READY_ATTEMPTS", 3)
    simctl.unready_lookups = 5
    with pytest.raises(RuntimeError, match="did not appear on simulator clone"):
        await device.clear_app(BUNDLE)


async def test_another_app_or_build_is_refused(simulator):
    device, _ = simulator
    await device.prepare_build(ios_build())
    with pytest.raises(RuntimeError, match="not the installed wallet"):
        await device.clear_app("org.other.wallet")
    with pytest.raises(ValueError, match="installed wallet changed"):
        await device.verify_build(BUNDLE, "0" * 64)
    with pytest.raises(ValueError, match="only iOS builds"):
        await device.prepare_build(ios_build("android"))


async def test_a_running_base_simulator_is_refused(simulator):
    device, simctl = simulator
    simctl.devices[BASE]["state"] = "Booted"
    with pytest.raises(RuntimeError, match="shut down the base simulator"):
        await device.prepare_build(ios_build())
    assert [entry[0] for entry in simctl.log] == ["list"]


async def test_cleanup_and_stale_clone_removal_keep_the_base(simulator):
    device, simctl = simulator
    await device.prepare_build(ios_build())
    await device.cleanup()
    assert set(simctl.devices) == {BASE}
    assert device.serial == ""

    simctl.devices["ABCDEF00-0000-0000-0000-000000000000"] = {
        "name": f"credimi-action-author-{BASE}-clone-9",
        "state": "Booted",
        "apps": set(),
    }
    simctl.devices["ABCDEF01-0000-0000-0000-000000000000"] = {
        "name": "someone else's simulator",
        "state": "Booted",
        "apps": set(),
    }
    assert await device.remove_stale_clones() == ["ABCDEF00-0000-0000-0000-000000000000"]
    assert set(simctl.devices) == {BASE, "ABCDEF01-0000-0000-0000-000000000000"}
