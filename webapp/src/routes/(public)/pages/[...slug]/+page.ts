// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getContentBySlug } from '$lib/content';
import tagsIndex from '$lib/content/tags-list.generated.json';

import type { EntryGenerator } from './$types';

const DEFAULT_SITE_URL = 'https://credimi.io';

export const entries: EntryGenerator = () => {
	const slugs = [...new Set(Object.values(tagsIndex).flat())];

	return slugs.map((slug) => ({ slug }));
};

function getSiteOrigin(url: URL) {
	if (url.origin.startsWith('http://sveltekit-prerender')) return DEFAULT_SITE_URL;
	return url.origin;
}

export async function load({ params, fetch, url }) {
	const content = await getContentBySlug(params.slug, fetch);
	if (!content) {
		error(404);
	}

	const origin = getSiteOrigin(url);
	const canonicalUrl = new URL(url.pathname, origin).toString();
	const socialImageUrl = new URL(content.attributes.socialCard ?? '/hero.png', origin).toString();

	return {
		...content,
		seo: {
			title: `${content.attributes.title} | Credimi`,
			description: content.attributes.description ?? '',
			canonicalUrl,
			socialImageUrl,
			keywords: content.attributes.tags.join(', '),
			publishedTime: content.attributes.date.toISOString(),
			modifiedTime: content.attributes.updatedOn.toISOString()
		}
	};
}
