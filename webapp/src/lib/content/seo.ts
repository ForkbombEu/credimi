// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { absoluteUrl, SITE_ORIGIN, type PageSeo } from '$lib/seo';

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
		imageAlt: content.attributes.socialCardAlt,
		type: 'article',
		article: { publishedTime, modifiedTime, tags },
		locales: content.locales,
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
			},
			{
				'@context': 'https://schema.org',
				'@type': 'BreadcrumbList',
				itemListElement: [
					['Home', '/'],
					['Guides', '/pages/guides'],
					[title, `/pages/${content.slug}`]
				].map(([name, path], index) => ({
					'@type': 'ListItem',
					position: index + 1,
					name,
					item: absoluteUrl(path, SITE_ORIGIN)
				}))
			},
			...(content.faq.length
				? [
						{
							'@context': 'https://schema.org',
							'@type': 'FAQPage',
							mainEntity: content.faq.map(({ question, answer }) => ({
								'@type': 'Question',
								name: question,
								acceptedAnswer: { '@type': 'Answer', text: answer }
							}))
						}
					]
				: [])
		]
	};
}
