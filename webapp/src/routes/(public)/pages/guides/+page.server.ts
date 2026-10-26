// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getContentBySlug } from '$lib/content/pages.server';
import tagsIndex from '$lib/content/tags-list.generated.json';
import { staticPages } from '$lib/seo';

async function loadPages(slugs: string[]) {
	const pages = await Promise.all(slugs.map((slug) => getContentBySlug(slug)));
	return pages.filter((page) => page !== undefined);
}

export async function load() {
	// Pages tagged `guide` by title, then the comparison and explainer that have their own routes.
	const guides = (await loadPages(tagsIndex.guide)).sort((a, b) =>
		a.attributes.title.localeCompare(b.attributes.title)
	);
	const explainers = await loadPages(['why-credimi', 'conformance-automation']);

	return {
		guides: [...guides, ...explainers].map(({ slug, attributes }) => ({
			href: `/pages/${slug}`,
			title: attributes.title,
			lead: attributes.lead ?? attributes.description,
			image: attributes.socialCard
		})),
		seo: staticPages['/pages/guides']
	};
}
