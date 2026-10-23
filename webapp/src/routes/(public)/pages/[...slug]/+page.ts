// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getContentBySlug } from '$lib/content';
import { getContentSeo } from '$lib/content/seo';
import tagsIndex from '$lib/content/tags-list.generated.json';

import type { EntryGenerator } from './$types';

// Served by their own routes under /pages; prerendering them here would collide with those routes.
const DEDICATED_ROUTE_SLUGS: Record<string, true> = {
	'why-credimi': true,
	'conformance-automation': true
};

// No markdown page is routed through here at the moment; 'auto' prerenders the ones listed in
// `entries` and renders any other slug on request (404 when it has no markdown).
export const prerender = 'auto';

export const entries: EntryGenerator = () => {
	const slugs = [...new Set(Object.values(tagsIndex).flat())];

	return slugs.filter((slug) => !DEDICATED_ROUTE_SLUGS[slug]).map((slug) => ({ slug }));
};

export async function load({ params, fetch }) {
	const content = await getContentBySlug(params.slug, fetch);
	if (!content) {
		error(404);
	}

	return { ...content, seo: getContentSeo(content) };
}
