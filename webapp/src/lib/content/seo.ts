// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { PageSeo } from '$lib/seo';

import type { ContentPage } from './types';

/** SEO for a markdown-backed page under /pages, from its frontmatter. */
export function getContentSeo(content: ContentPage): PageSeo {
	const { title, description, tags, date, updatedOn } = content.attributes;
	const publishedTime = date.toISOString();
	const modifiedTime = updatedOn.toISOString();

	return {
		title,
		description,
		image: content.attributes.socialCard,
		type: 'article',
		article: { publishedTime, modifiedTime, tags },
		jsonLd: [
			{
				'@context': 'https://schema.org',
				'@type': 'TechArticle',
				headline: title,
				description,
				datePublished: publishedTime,
				dateModified: modifiedTime,
				keywords: tags.join(', '),
				author: { '@type': 'Organization', name: 'Credimi', url: 'https://credimi.io' },
				publisher: {
					'@type': 'Organization',
					name: 'Forkbomb BV',
					url: 'https://forkbomb.eu'
				}
			}
		]
	};
}
