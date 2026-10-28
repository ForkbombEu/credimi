# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Cross-language contracts; credentials never enter the model's tool context."""

from typing import Literal

from pydantic import BaseModel, ConfigDict, Field

GO_ACTIVITY_QUEUE = "PipelineTaskQueue"
RESOLVE_BUILD = "Resolve action author build"
SAVE_ACTION = "Save authored wallet action"
PREPARE_BUILD = "action_author.prepare_build"


class AuthoringContext(BaseModel):
    model_config = ConfigDict(extra="forbid")
    wallet_version_identifier: str = Field(min_length=1)
    actor_id: str = Field(min_length=1)
    actor_collection: Literal["users", "_superusers"]
    slot: Literal["onboarding"]
    app_id: str = ""

    def build_payload(self) -> dict[str, str]:
        return self.model_dump(exclude={"slot", "app_id"})


class ResolvedBuild(BaseModel):
    wallet_id: str
    version_id: str
    version_identifier: str
    owner_namespace: str
    installer_name: str
    apk_sha256: str = Field(pattern=r"^[a-f0-9]{64}$")
    download_grant: str = Field(repr=False)


class InstalledBuild(BaseModel):
    package_id: str
    version_code: str
    version_name: str
    apk_sha256: str


class PrepareBuildRequest(BaseModel):
    build: ResolvedBuild
    expected_app_id: str = ""


class SaveActionApproval(BaseModel):
    """Confirmed internal save input and replay-compatible legacy handler contract."""

    model_config = ConfigDict(extra="forbid")
    name: str = Field(min_length=3)
    candidate_sha256: str = Field(pattern=r"^[a-f0-9]{64}$")
    confirm: bool = False
    replace_action_id: str = ""
    expected_existing_sha256: str = ""
