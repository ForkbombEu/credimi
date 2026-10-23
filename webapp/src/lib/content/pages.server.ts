// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { plainText } from '$lib/seo';
import fm from 'front-matter';
import { marked } from 'marked';

import { baseLocale, getLocale } from '@/i18n/paraglide/runtime';

import { pageFrontMatterSchema, type ContentPage } from './types';

export const URL_SEARCH_PARAM_NAME = 'tag';

// Bundled at build time: the server runs from a directory without `static/`, and fetching its own
// static files fails there, so runtime reads lost every page from the sitemap.
const markdownFiles = import.meta.glob<string>('/static/pages/**/*.md', {
	query: '?raw',
	import: 'default'
});

export async function getContentBySlug(slug: string): Promise<ContentPage | undefined> {
	const load =
		markdownFiles[`/static/pages/${slug}/${getLocale()}.md`] ??
		markdownFiles[`/static/pages/${slug}/${baseLocale}.md`];
	if (!load) return undefined;
	const { attributes, body } = fm(await load());

	const parsed = pageFrontMatterSchema.safeParse(attributes);
	if (!parsed.success) return undefined;

	// H2s get ids for the "On this page" links.
	const headings: ContentPage['headings'] = [];
	const withIds = marked(body, { async: false }).replace(/<h2>([\s\S]*?)<\/h2>/g, (_, inner) => {
		const text = plainText(inner, 200);
		const id = text
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-|-$/g, '');
		headings.push({ id, text });
		return `<h2 id="${id}">${inner}</h2>`;
	});

	// Each H3 under "FAQ" becomes a collapsible question, and feeds FAQPage structured data.
	const faq: ContentPage['faq'] = [];
	const html = withIds.replace(
		/(<h2 id="faq">[\s\S]*?<\/h2>)([\s\S]*?)(?=<h2[\s>]|<div class="content-related">|$)/,
		(_, heading: string, items: string) =>
			heading +
			items
				.split(/(?=<h3>)/)
				.map((item) => {
					const match = item.match(/^<h3>([\s\S]*?)<\/h3>([\s\S]*)$/);
					if (!match) return item;
					const [, question, answer] = match;
					faq.push({
						question: plainText(question, 300),
						answer: plainText(answer, 2000)
					});
					return `<details class="faq-item"><summary>${question}</summary>${answer}</details>`;
				})
				.join('')
	);

	return { attributes: parsed.data, body: html, slug, headings, faq };
}
