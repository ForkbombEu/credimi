// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { plainText } from '$lib/seo';

import { notFoundAs404 } from '@/pocketbase/index.js';
import { PocketbaseQueryAgent } from '@/pocketbase/query/agent.js';

export const load = async ({ params, fetch }) => {
	const news = await new PocketbaseQueryAgent({ collection: 'news' }, { fetch })
		.getOne(params.news_id)
		.catch(notFoundAs404);

	return {
		news,
		seo: {
			title: news.title,
			description: plainText(news.summary),
			type: 'article' as const,
			article: { publishedTime: news.created, modifiedTime: news.updated, tags: [] },
			jsonLd: [
				{
					'@context': 'https://schema.org',
					'@type': 'NewsArticle',
					headline: news.title,
					description: plainText(news.summary, 300),
					datePublished: news.created,
					dateModified: news.updated,
					publisher: { '@type': 'Organization', name: 'Credimi' }
				}
			]
		}
	};
};
