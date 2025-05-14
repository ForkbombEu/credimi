// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { loadFeatureFlags } from '@/features';
import { error } from '@sveltejs/kit';

export const load = async ({ fetch }) => {
	if (!(await loadFeatureFlags(fetch)).ORGANIZATIONS) error(404);
};
