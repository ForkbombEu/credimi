// SPDX-FileCopyrightText: 2025 Forkbomb BV

// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getContentBySlug } from '$lib/content';

//

export async function load({ params }) {
	const content = await getContentBySlug(params.slug);
	if (!content) {
		error(404);
	}
	return {
		content,
		slug: params.slug
	};
}
