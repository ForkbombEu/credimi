// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { plainText } from '$lib/seo';
import fm from 'front-matter';
import { marked } from 'marked';
import { codeToHtml, type BundledLanguage } from 'shiki';

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

	// H2s and H3s get ids and a "§" link, so any section can be shared; H2s also feed
	// the "On this page" bar.
	const headings: ContentPage['headings'] = [];
	const usedIds = new Set<string>();
	const withIds = marked(body, { async: false }).replace(
		/<h([23])>([\s\S]*?)<\/h\1>/g,
		(_, level: string, inner: string) => {
			const text = plainText(inner, 200);
			const base = text
				.toLowerCase()
				.replace(/[^a-z0-9]+/g, '-')
				.replace(/^-|-$/g, '');
			let id = base;
			for (let n = 2; usedIds.has(id); n++) id = `${base}-${n}`;
			usedIds.add(id);
			if (level === '2') headings.push({ id, text });
			return `<h${level} id="${id}">${inner}<a class="heading-anchor" href="#${id}" aria-label="Link to this section">§</a></h${level}>`;
		}
	);

	// Each H3 under "FAQ" becomes a collapsible question, and feeds FAQPage structured data.
	const faq: ContentPage['faq'] = [];
	const html = withIds.replace(
		/(<h2 id="faq">[\s\S]*?<\/h2>)([\s\S]*?)(?=<h2[\s>]|<div class="content-related">|$)/,
		(_, heading: string, items: string) =>
			heading +
			items
				.split(/(?=<h3[\s>])/)
				.map((item) => {
					const match = item.match(
						/^<h3 id="([^"]+)">([\s\S]*?)(<a class="heading-anchor"[\s\S]*?<\/a>)<\/h3>([\s\S]*)$/
					);
					if (!match) return item;
					const [, id, question, anchor, answer] = match;
					faq.push({
						question: plainText(question, 300),
						answer: plainText(answer, 2000)
					});
					return `<details class="faq-item" id="${id}"><summary>${question}${anchor}</summary>${answer}</details>`;
				})
				.join('')
	);

	// Code blocks get the platform's Shiki highlighting here, on the server: no highlighter
	// reaches the browser for a guide page.
	let highlighted = html;
	for (const [block, lang, code] of html.matchAll(
		/<pre><code class="language-([\w-]+)">([\s\S]*?)<\/code><\/pre>/g
	)) {
		const source = code
			.replace(/&lt;/g, '<')
			.replace(/&gt;/g, '>')
			.replace(/&quot;/g, '"')
			.replace(/&#39;/g, "'")
			.replace(/&amp;/g, '&')
			.trimEnd();
		const rendered = await codeToHtml(source, {
			lang: lang as BundledLanguage,
			theme: 'github-light'
		}).catch(() => block);
		highlighted = highlighted.replace(block, () => rendered);
	}

	const locales = Object.keys(markdownFiles)
		.filter((path) => path.startsWith(`/static/pages/${slug}/`))
		.map((path) => path.slice(path.lastIndexOf('/') + 1, -'.md'.length));

	return { attributes: parsed.data, body: highlighted, slug, headings, faq, locales };
}
