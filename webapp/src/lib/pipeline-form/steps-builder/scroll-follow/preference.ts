// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { lsSync } from 'rune-sync/localstorage';

const STORAGE_KEY = 'pipeline_composer_scroll_follow';

/** Pipeline Composer scroll-follow toggle. Default on when unset. */
export const scrollFollowPreference = lsSync<{ enabled: boolean }>(STORAGE_KEY, {
	enabled: true
});
