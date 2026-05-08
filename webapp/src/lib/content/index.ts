// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { browser } from '$app/environment';
import fm from 'front-matter';
import { marked } from 'marked';

import { baseLocale, getLocale } from '@/i18n/paraglide/runtime';

import { pageFrontMatterSchema, type ContentPage } from './types';

export const URL_SEARCH_PARAM_NAME = 'tag';

async function loadMarkdownFile(
	pathname: string,
	fetcher: typeof fetch
): Promise<string | undefined> {
	const response = await fetcher(pathname).catch(() => undefined);
	if (response?.ok) return response.text();

	if (browser || !pathname.startsWith('/pages/')) return undefined;

	const relativePath = pathname.slice('/pages/'.length);

	try {
		const [{ readFile }, path] = await Promise.all([
			import('node:fs/promises'),
			import('node:path')
		]);

		const normalizedRelativePath = path.posix.normalize(relativePath);
		if (
			relativePath.includes('\\') ||
			normalizedRelativePath.startsWith('../') ||
			normalizedRelativePath === '..' ||
			normalizedRelativePath.startsWith('/')
		) {
			return undefined;
		}

		const pagesRoot = path.resolve(process.cwd(), 'static/pages');
		const fullPath = path.resolve(pagesRoot, normalizedRelativePath);
		const relativeToRoot = path.relative(pagesRoot, fullPath);

		if (relativeToRoot.startsWith('..') || path.isAbsolute(relativeToRoot)) {
			return undefined;
		}

		return await readFile(fullPath, 'utf8');
	} catch {
		return undefined;
	}
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
