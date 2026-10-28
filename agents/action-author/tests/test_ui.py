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


async def test_harness_landing_page_opens_authenticated_start():
    app = create_app()
    async with httpx.AsyncClient(
        transport=httpx.ASGITransport(app=app), base_url="http://harness"
    ) as client:
        response = await client.get("/")
    assert response.status_code == 307
    assert response.headers["location"] == "/start"


async def test_start_page_is_not_cached_or_embeddable():
    app = create_app()
    async with httpx.AsyncClient(
        transport=httpx.ASGITransport(app=app), base_url="http://harness"
    ) as client:
        response = await client.get("/start")
    assert response.status_code == 200
    assert response.headers["cache-control"] == "no-store"
    assert "frame-ancestors 'none'" in response.headers["content-security-policy"]
