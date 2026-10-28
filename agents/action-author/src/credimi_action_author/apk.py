# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Download a single authorized build, inspect it, and install its verified bytes."""

import hashlib
import os
import re
from pathlib import Path
from tempfile import TemporaryDirectory
from urllib.parse import urlsplit

import httpx

from .contracts import InstalledBuild, PrepareBuildRequest
from .device import _run
from .validation import APP_ID

MAX_APK_BYTES = 500 * 1024 * 1024


def credimi_url() -> str:
    base = os.environ.get("ACTION_AUTHOR_CREDIMI_URL", "http://localhost:8090").rstrip("/")
    parsed = urlsplit(base)
    if (
        parsed.scheme not in {"http", "https"}
        or not parsed.hostname
        or parsed.username
        or parsed.password
        or parsed.query
        or parsed.fragment
        or parsed.path
    ):
        raise ValueError("ACTION_AUTHOR_CREDIMI_URL must be a trusted HTTP(S) origin")
    return base


async def download_apk(req: PrepareBuildRequest, destination: Path) -> None:
    digest = hashlib.sha256()
    size = 0
    async with httpx.AsyncClient(
        timeout=httpx.Timeout(120, connect=10), follow_redirects=False, trust_env=False
    ) as client:
        async with client.stream(
            "GET",
            credimi_url() + "/api/action-author/installer",
            headers={"Credimi-Artifact-Grant": req.build.download_grant},
        ) as response:
            response.raise_for_status()
            with destination.open("wb") as stream:
                async for chunk in response.aiter_bytes(64 * 1024):
                    size += len(chunk)
                    if size > MAX_APK_BYTES:
                        raise ValueError("selected APK exceeds the 500 MiB installer limit")
                    digest.update(chunk)
                    stream.write(chunk)
    if digest.hexdigest() != req.build.apk_sha256:
        raise ValueError("selected APK SHA256 does not match the resolved Credimi build")


async def _command(arguments: list[str], timeout: float = 60) -> str:
    code, output = await _run(arguments, timeout)
    if code:
        raise RuntimeError(f"{arguments[0]} failed: {output.strip()}")
    return output.strip()


async def installed_digest(serial: str, package_id: str) -> str:
    if not APP_ID.fullmatch(package_id):
        raise ValueError("invalid selected APK package ID")
    paths = await _command(["adb", "-s", serial, "shell", "pm", "path", package_id])
    files = [
        line.removeprefix("package:") for line in paths.splitlines() if line.startswith("package:")
    ]
    # The selected installer is one APK, not an APK set. Do not claim that a
    # separately installed split package is the same validated build.
    if (
        len(files) != 1
        or not files[0].startswith("/data/app/")
        or any(c.isspace() for c in files[0])
    ):
        raise ValueError("installed wallet is missing or is not the selected single APK")
    output = await _command(["adb", "-s", serial, "shell", "sha256sum", files[0]])
    match = re.match(r"^([a-f0-9]{64})\s", output)
    if not match:
        raise ValueError("cannot verify installed APK SHA256")
    return match.group(1)


async def prepare_apk(device, req: PrepareBuildRequest) -> InstalledBuild:
    with TemporaryDirectory(prefix="credimi-author-apk-") as root:
        installer = Path(root) / "selected.apk"
        await download_apk(req, installer)
        package_id = await _command(["apkanalyzer", "manifest", "application-id", str(installer)])
        if not APP_ID.fullmatch(package_id):
            raise ValueError("selected APK has an invalid Android package ID")
        if req.expected_app_id and req.expected_app_id != package_id:
            raise ValueError("requested package does not match the selected Credimi APK")
        version_code = await _command(["apkanalyzer", "manifest", "version-code", str(installer)])
        version_name = await _command(["apkanalyzer", "manifest", "version-name", str(installer)])
        await device.close_session()
        await _command(["adb", "-s", device.serial, "install", "-r", str(installer)], 180)
        if await installed_digest(device.serial, package_id) != req.build.apk_sha256:
            raise ValueError("installed wallet does not match the selected Credimi APK")
        return InstalledBuild(
            package_id=package_id,
            version_code=version_code,
            version_name=version_name,
            apk_sha256=req.build.apk_sha256,
        )
