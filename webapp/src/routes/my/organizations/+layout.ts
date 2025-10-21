// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';

import { loadFeatureFlags } from '@/features';

export const load = async ({ fetch }) => {
	error(404);
	if (!(await loadFeatureFlags(fetch)).ORGANIZATIONS) error(404);
};
