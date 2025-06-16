// SPDX-FileCopyrightText: 2025 Forkbomb BV

// SPDX-License-Identifier: AGPL-3.0-or-later

import { getContentBySlug } from '$lib/content';
import { error } from '@sveltejs/kit';

export async function load({ params }) {
	return (await getContentBySlug(params.slug)) ?? error(404);
}
