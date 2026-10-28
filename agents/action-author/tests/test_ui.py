# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

import httpx

from credimi_action_author.ui import create_app


async def test_harness_cannot_start_sessions_with_caller_supplied_actor():
    app = create_app()
    async with httpx.AsyncClient(
        transport=httpx.ASGITransport(app=app), base_url="http://harness"
    ) as client:
        response = await client.post(
            "/api/sessions",
            json={
                "agent_workflow_type": "ActionAuthor",
                "data": {
                    "actor_id": "someone-else",
                    "actor_collection": "_superusers",
                    "wallet_version_identifier": "other/wallet/build",
                    "slot": "onboarding",
                },
            },
        )
    assert response.status_code == 403
