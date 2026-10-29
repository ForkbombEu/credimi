# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Inspect the selected APK and install its verified bytes on the Android phone."""

import re
from pathlib import Path
from tempfile import TemporaryDirectory

from .contracts import InstalledBuild, PrepareBuildRequest
from .installer import command, download_installer
from .validation import APP_ID


async def installed_digest(serial: str, package_id: str) -> str:
    if not APP_ID.fullmatch(package_id):
        raise ValueError("invalid selected APK package ID")
    paths = await command(["adb", "-s", serial, "shell", "pm", "path", package_id])
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
    output = await command(["adb", "-s", serial, "shell", "sha256sum", files[0]])
    match = re.match(r"^([a-f0-9]{64})\s", output)
    if not match:
        raise ValueError("cannot verify installed APK SHA256")
    return match.group(1)


async def prepare_apk(device, req: PrepareBuildRequest) -> InstalledBuild:
    if req.build.platform != "android":
        raise ValueError("an Android phone installs only Android builds")
    with TemporaryDirectory(prefix="credimi-author-apk-") as root:
        installer = Path(root) / "selected.apk"
        await download_installer(req, installer)
        package_id = await command(["apkanalyzer", "manifest", "application-id", str(installer)])
        if not APP_ID.fullmatch(package_id):
            raise ValueError("selected APK has an invalid Android package ID")
        if req.expected_app_id and req.expected_app_id != package_id:
            raise ValueError("requested package does not match the selected Credimi APK")
        version_code = await command(["apkanalyzer", "manifest", "version-code", str(installer)])
        version_name = await command(["apkanalyzer", "manifest", "version-name", str(installer)])
        await device.close_session()
        await command(["adb", "-s", device.serial, "install", "-r", str(installer)], 180)
        if await installed_digest(device.serial, package_id) != req.build.installer_sha256:
            raise ValueError("installed wallet does not match the selected Credimi APK")
        return InstalledBuild(
            package_id=package_id,
            version_code=version_code,
            version_name=version_name,
            installer_sha256=req.build.installer_sha256,
        )
