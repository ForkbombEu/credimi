// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { absoluteUrl, localeAlternates, SITE_ORIGIN } from '$lib/seo';
import { listPublicPages } from '$lib/seo/public-pages';

const xmlEscapes: Record<string, string> = {
	'&': '&amp;',
	'<': '&lt;',
	'>': '&gt;',
	'"': '&quot;',
	"'": '&apos;'
};

/** Sitemap of every public page, each with its locale variants as hreflang alternates. */
export const GET = async ({ fetch }) => {
	const origin = SITE_ORIGIN;
	const pages = await listPublicPages(fetch);
	const escape = (value: string) => value.replace(/[&<>"']/g, (char) => xmlEscapes[char]);

	const urls = pages.map((page) => {
		const alternates = localeAlternates(page.path, origin)
			.map(
				(alternate) =>
					`<xhtml:link rel="alternate" hreflang="${alternate.hreflang}" href="${escape(alternate.href)}"/>`
			)
			.join('');
		const lastmod = page.lastmod ? `<lastmod>${page.lastmod}</lastmod>` : '';
		return `<url><loc>${escape(absoluteUrl(page.path, origin))}</loc>${lastmod}${alternates}</url>`;
	});

	const xml = [
		'<?xml version="1.0" encoding="UTF-8"?>',
		'<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">',
		...urls,
		'</urlset>'
	].join('\n');

	return new Response(xml, {
		headers: {
			'content-type': 'application/xml; charset=utf-8',
			'cache-control': 'public, max-age=3600'
		}
	});
};
