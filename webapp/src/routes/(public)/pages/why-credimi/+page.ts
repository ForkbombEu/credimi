// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getContentBySlug } from '$lib/content';
import { getContentSeo } from '$lib/content/seo';

// static/pages/why-credimi/en.md keeps the metadata (tags, SEO, social card); the page body lives here.
export async function load({ fetch, url }) {
	const content = await getContentBySlug('why-credimi', fetch);
	if (!content) {
		error(404);
	}

	return { tags: content.attributes.tags, seo: getContentSeo(content, url) };
}
