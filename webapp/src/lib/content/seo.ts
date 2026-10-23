// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ContentPage } from './types';

const DEFAULT_SITE_URL = 'https://credimi.io';

function getSiteOrigin(url: URL) {
	if (url.origin.startsWith('http://sveltekit-prerender')) return DEFAULT_SITE_URL;
	return url.origin;
}

export type ContentSeo = {
	title: string;
	description: string;
	canonicalUrl: string;
	socialImageUrl: string;
	keywords: string;
	publishedTime: string;
	modifiedTime: string;
};

export function getContentSeo(content: ContentPage, url: URL): ContentSeo {
	const origin = getSiteOrigin(url);

	return {
		title: `${content.attributes.title} | Credimi`,
		description: content.attributes.description ?? '',
		canonicalUrl: new URL(url.pathname, origin).toString(),
		socialImageUrl: new URL(content.attributes.socialCard ?? '/hero.png', origin).toString(),
		keywords: content.attributes.tags.join(', '),
		publishedTime: content.attributes.date.toISOString(),
		modifiedTime: content.attributes.updatedOn.toISOString()
	};
}
