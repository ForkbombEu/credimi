# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""An iOS simulator on this Mac, driven by agent-device, ``xcrun simctl`` and maestro.

Credimi stores iOS builds as zipped simulator ``.app`` bundles. For each session the worker:

1. downloads the selected zip, checks its SHA256, extracts it with ``ditto`` (which keeps file
   modes and symlinks) and finds its single ``.app``, at the top level or under ``Payload/``, as
   the Credimi runner does;
2. clones the configured base simulator into a session template, installs the ``.app`` there and
   shuts the template down. The base simulator is never booted or modified;
3. on every reset (``reset_app`` and before each replay) deletes the current clone and boots a
   new clone of the template. App data, keychain and permission grants therefore start as freshly
   installed, the isolation pipelines get from their per-run simulator clone.

``simctl`` has no equivalent of Android's ``pm clear``, and an uninstall keeps keychain items, so
a new clone is the reset. The installed bundle cannot be hashed back to the uploaded zip; the
worker binds the template to the bundle ID and digest it installed and checks that the clone
still has that bundle.
"""

from __future__ import annotations

import asyncio
import json
import re
import zipfile
from pathlib import Path, PurePosixPath
from tempfile import TemporaryDirectory

from .contracts import InstalledBuild, PrepareBuildRequest
from .device import (
    _run,
    close_agent_device,
    maestro_mcp_call,
    require_tools,
)
from .installer import command, download_installer
from .validation import APP_ID

CLONE_PREFIX = "credimi-action-author"
BOOT_TIMEOUT = 300.0
INSTALL_TIMEOUT = 300.0
# A booted clone lists the template's apps within seconds; poll before using it.
APP_READY_ATTEMPTS = 30
APP_READY_INTERVAL = 1.0
_UDID = re.compile(r"[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}")
SIMULATOR_PLATFORM = "iPhoneSimulator"


def app_root(names: list[str]) -> PurePosixPath:
    """The single ``.app`` bundle of a zip's member names, at any depth (e.g. ``Payload/``).

    Rejects absolute and parent-relative members, which ``ditto`` would write outside the
    extraction directory.
    """
    roots = set()
    for name in names:
        path = PurePosixPath(name)
        if path.is_absolute() or ".." in path.parts:
            raise ValueError(f"selected iOS installer has an unsafe path: {name!r}")
        if path.parts and path.parts[0] == "__MACOSX":
            continue
        for index, part in enumerate(path.parts):
            if part.lower().endswith(".app"):
                roots.add(PurePosixPath(*path.parts[: index + 1]))
                break
    if len(roots) != 1:
        raise ValueError(
            f"selected iOS installer must contain exactly one .app bundle, found {len(roots)}"
        )
    return roots.pop()


async def extract_app(archive: Path, destination: Path) -> Path:
    try:
        with zipfile.ZipFile(archive) as bundle:
            root = app_root(bundle.namelist())
    except zipfile.BadZipFile as exc:
        raise ValueError("selected iOS installer is not a zip archive") from exc
    destination.mkdir(parents=True, exist_ok=True)
    await command(["ditto", "-x", "-k", str(archive), str(destination)], 300)
    app = destination.joinpath(*root.parts)
    if not (app / "Info.plist").is_file():
        raise ValueError(f"selected iOS app {root} has no Info.plist")
    return app


async def _plist_value(plist: Path, key: str, kind: str = "raw") -> str:
    code, output = await _run(["plutil", "-extract", key, kind, "-o", "-", str(plist)], 30)
    return output.strip() if code == 0 else ""


async def read_app_info(app: Path) -> tuple[str, str, str]:
    """(bundle ID, CFBundleVersion, CFBundleShortVersionString) of a simulator ``.app``."""
    plist = app / "Info.plist"
    bundle_id = await _plist_value(plist, "CFBundleIdentifier")
    if not APP_ID.fullmatch(bundle_id):
        raise ValueError("selected iOS app has an invalid bundle ID")
    raw_platforms = await _plist_value(plist, "CFBundleSupportedPlatforms", "json")
    try:
        platforms = json.loads(raw_platforms) if raw_platforms else []
    except ValueError:
        platforms = []
    if SIMULATOR_PLATFORM not in platforms:
        raise ValueError(
            f"selected iOS app is not a simulator build (supported platforms: {platforms}); "
            "upload a zipped simulator .app"
        )
    return (
        bundle_id,
        await _plist_value(plist, "CFBundleVersion"),
        await _plist_value(plist, "CFBundleShortVersionString"),
    )


class IOSSimulatorDevice:
    """Clones of the base simulator ``base_udid``: one session template, one current clone."""

    platform = "ios"

    def __init__(self, base_udid: str) -> None:
        if not _UDID.fullmatch(base_udid):
            raise ValueError(f"invalid iOS simulator UDID {base_udid!r}")
        require_tools("agent-device", "maestro", "xcrun", "plutil", "ditto")
        self.base_udid = base_udid
        self.configured_device = base_udid
        self._prefix = f"{CLONE_PREFIX}-{base_udid}"
        self._template = ""
        self._clone = ""
        self._clones = 0
        self._installed: tuple[str, str] | None = None
        self._lock = asyncio.Lock()

    @property
    def serial(self) -> str:
        """The current clone: the simulator agent-device and Maestro drive."""
        return self._clone

    async def _devices(self) -> list[dict]:
        listed = json.loads(await command(["xcrun", "simctl", "list", "devices", "--json"]))
        return [device for devices in listed["devices"].values() for device in devices]

    async def check_base(self) -> None:
        base = next((d for d in await self._devices() if d.get("udid") == self.base_udid), None)
        if base is None:
            raise RuntimeError(f"iOS simulator {self.base_udid} not found (xcrun simctl list)")
        if base.get("state") != "Shutdown":
            raise RuntimeError(
                f"shut down the base simulator {base.get('name')} ({self.base_udid}): "
                "the worker clones it and never drives it directly"
            )

    async def remove_stale_clones(self) -> list[str]:
        """Delete clones left by an earlier worker of this base simulator."""
        stale = [
            d["udid"] for d in await self._devices() if d.get("name", "").startswith(self._prefix)
        ]
        for udid in stale:
            await self._delete(udid)
        return stale

    async def _delete(self, udid: str) -> None:
        await _run(["xcrun", "simctl", "shutdown", udid], 120)
        await command(["xcrun", "simctl", "delete", udid], 120)

    async def _clone_booted(self, source: str, name: str) -> str:
        udid = await command(["xcrun", "simctl", "clone", source, f"{self._prefix}-{name}"], 300)
        udid = udid.splitlines()[-1].strip() if udid else ""
        if not _UDID.fullmatch(udid):
            raise RuntimeError(f"simctl clone returned no UDID: {udid!r}")
        try:
            await command(["xcrun", "simctl", "boot", udid], 120)
            await command(["xcrun", "simctl", "bootstatus", udid, "-b"], BOOT_TIMEOUT)
        except Exception:
            await self._delete(udid)
            raise
        return udid

    async def _drop_clone(self) -> None:
        clone, self._clone = self._clone, ""
        if clone:
            await self._delete(clone)

    async def _new_clone(self) -> str:
        await self._drop_clone()
        self._clones += 1
        self._clone = await self._clone_booted(self._template, f"clone-{self._clones}")
        await self._wait_for_app(self._clone)
        return self._clone

    async def _wait_for_app(self, udid: str) -> None:
        """A clone can answer `bootstatus` before its app registry lists the wallet."""
        if self._installed is None:
            return
        out = ""
        for _ in range(APP_READY_ATTEMPTS):
            code, out = await _run(
                ["xcrun", "simctl", "get_app_container", udid, self._installed[0]], 30
            )
            if code == 0:
                return
            await asyncio.sleep(APP_READY_INTERVAL)
        raise RuntimeError(f"the wallet did not appear on simulator clone {udid}: {out.strip()}")

    async def _discard(self) -> None:
        self._installed = None
        await self._drop_clone()
        template, self._template = self._template, ""
        if template:
            await self._delete(template)

    async def cleanup(self) -> None:
        async with self._lock:
            await self._discard()

    async def close_session(self) -> str:
        if not self._clone:
            return ""
        return await close_agent_device(["--udid", self._clone, "--platform", "ios"])

    async def clear_app(self, app_id: str) -> str:
        if self._installed is None or self._installed[0] != app_id:
            raise RuntimeError(f"{app_id} is not the installed wallet")
        async with self._lock:
            udid = await self._new_clone()
        return f"fresh simulator clone {udid}"

    def _require_clone(self) -> str:
        if not self._clone:
            raise RuntimeError("no simulator clone is running; call reset_app")
        return self._clone

    async def maestro_test(self, flow: Path, timeout: float) -> tuple[int, str]:
        cmd = ["maestro", "--device", self._require_clone(), "test", str(flow)]
        # Screenshots land in the flow's (temporary) directory.
        return await _run(cmd, timeout, cwd=flow.parent)

    async def inspect_screen(self) -> str:
        return await maestro_mcp_call("inspect_screen", {"device_id": self._require_clone()})

    async def cheat_sheet(self) -> str:
        return await maestro_mcp_call("cheat_sheet", {})

    async def prepare_build(self, req: PrepareBuildRequest) -> InstalledBuild:
        if req.build.platform != "ios":
            raise ValueError("an iOS simulator installs only iOS builds")
        async with self._lock:
            await self.check_base()
            with TemporaryDirectory(prefix="credimi-author-ios-") as root:
                archive = Path(root) / "selected.zip"
                await download_installer(req, archive)
                app = await extract_app(archive, Path(root) / "app")
                bundle_id, version_code, version_name = await read_app_info(app)
                if req.expected_app_id and req.expected_app_id != bundle_id:
                    raise ValueError("requested bundle ID does not match the selected iOS app")
                if self._clone:
                    await self.close_session()
                await self._discard()
                self._template = await self._clone_booted(self.base_udid, "template")
                await command(
                    ["xcrun", "simctl", "install", self._template, str(app)], INSTALL_TIMEOUT
                )
                await command(["xcrun", "simctl", "get_app_container", self._template, bundle_id])
                await command(["xcrun", "simctl", "shutdown", self._template], 120)
            self._installed = (bundle_id, req.build.installer_sha256)
            await self._new_clone()
        return InstalledBuild(
            package_id=bundle_id,
            version_code=version_code,
            version_name=version_name,
            installer_sha256=req.build.installer_sha256,
        )

    async def verify_build(self, app_id: str, installer_sha256: str) -> None:
        if not self._template or self._installed != (app_id, installer_sha256):
            raise ValueError("installed wallet changed since the Credimi build was selected")
        if self._clone:
            code, out = await _run(
                ["xcrun", "simctl", "get_app_container", self._clone, app_id], 60
            )
            if code != 0:
                raise ValueError(f"the wallet is missing from the simulator clone: {out.strip()}")
