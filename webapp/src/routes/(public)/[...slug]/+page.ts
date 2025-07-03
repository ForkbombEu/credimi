// SPDX-FileCopyrightText: 2025 Forkbomb BV

// SPDX-License-Identifier: AGPL-3.0-or-later

import { getContentBySlug } from '$lib/content';
import { error } from '@sveltejs/kit';

export async function load({ params }) {
	const content = await getContentBySlug(params.slug);
	if (!content) {
		error(404);
	}
	return content;
}
