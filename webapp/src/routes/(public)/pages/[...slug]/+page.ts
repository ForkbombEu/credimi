// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getContentBySlug } from '$lib/content';
import { getContentSeo } from '$lib/content/seo';
import tagsIndex from '$lib/content/tags-list.generated.json';

import type { EntryGenerator } from './$types';

// Served by their own routes under /pages; prerendering them here would collide with those routes.
const DEDICATED_ROUTE_SLUGS = new Set(['why-credimi', 'conformance-automation']);

export const entries: EntryGenerator = () => {
	const slugs = [...new Set(Object.values(tagsIndex).flat())];

	return slugs.filter((slug) => !DEDICATED_ROUTE_SLUGS.has(slug)).map((slug) => ({ slug }));
};

export async function load({ params, fetch, url }) {
	const content = await getContentBySlug(params.slug, fetch);
	if (!content) {
		error(404);
	}

	return { ...content, seo: getContentSeo(content, url) };
}
