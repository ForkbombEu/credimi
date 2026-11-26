// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getContentBySlug } from '$lib/content/pages.server';
import { getContentSeo } from '$lib/content/seo';

// static/pages/extras/en.md keeps the metadata and the FAQ; the rest of the page body lives here.
export async function load() {
	const content = await getContentBySlug('extras');
	if (!content) {
		error(404);
	}

	return { seo: getContentSeo(content), faq: content.faq };
}
