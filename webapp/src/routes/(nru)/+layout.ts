// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';

import { verifyUser } from '@/auth/verifyUser';
import { loadFeatureFlags } from '@/features';
import { redirect } from '@/i18n';

export const load = async ({ fetch }) => {
	if (!(await loadFeatureFlags(fetch)).AUTH) error(404);
	if (await verifyUser(fetch)) redirect('/my');
};
