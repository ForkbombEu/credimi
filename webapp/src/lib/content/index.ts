// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import fm from 'front-matter';
import { marked } from 'marked';
import { browser } from '$app/environment';

import { baseLocale, getLocale } from '@/i18n/paraglide/runtime';

import { pageFrontMatterSchema, type ContentPage } from './types';

export const URL_SEARCH_PARAM_NAME = 'tag';

async function loadMarkdownFile(
	pathname: string,
	fetcher: typeof fetch
): Promise<string | undefined> {
	if (!browser) {
		const relativePath = pathname.replace(/^\/pages\//, '');

		try {
			const [{ readFile }, path] = await Promise.all([
				import('node:fs/promises'),
				import('node:path')
			]);
			const fullPath = path.resolve(process.cwd(), 'static/pages', relativePath);

			return await readFile(fullPath, 'utf8');
		} catch {
			return undefined;
		}
	}

	const response = await fetcher(pathname);
	if (!response.ok) return undefined;
	return response.text();
}

export async function getContentBySlug(
	slug: string,
	fetcher: typeof fetch = fetch
): Promise<ContentPage | undefined> {
	const locale = getLocale();
	const fallbackLocale = baseLocale;
	const raw =
		(await loadMarkdownFile(`/pages/${slug}/${locale}.md`, fetcher)) ??
		(await loadMarkdownFile(`/pages/${slug}/${fallbackLocale}.md`, fetcher));
	if (!raw) return undefined;
	const { attributes, body } = fm(raw);

	const parsed = pageFrontMatterSchema.safeParse(attributes);
	if (!parsed.success) return undefined;

	return {
		attributes: parsed.data,
		body: body ? marked(body, { async: false }) : '',
		slug
	};
}
