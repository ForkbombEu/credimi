# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

from datetime import date

from credimi_news_agent.daily import next_window

TODAY = date(2026, 10, 23)
YESTERDAY = date(2026, 10, 22)


def test_first_run_scans_yesterday():
    assert next_window(TODAY, None) == (YESTERDAY, YESTERDAY)


def test_cursor_three_days_back_scans_the_two_missed_days():
    assert next_window(TODAY, "2026-10-20") == (date(2026, 10, 21), YESTERDAY)


def test_cursor_at_yesterday_has_nothing_to_scan():
    assert next_window(TODAY, "2026-10-22") is None


def test_old_cursor_is_clamped_to_the_longest_window():
    since, until = next_window(TODAY, "2026-04-06")
    assert until == YESTERDAY
    assert (until - since).days + 1 == 92
