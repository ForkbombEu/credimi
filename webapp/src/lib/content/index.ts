// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { baseLocale, getLocale } from '@/i18n/paraglide/runtime';
import fm from 'front-matter';
import { pageFrontMatterSchema, type ContentPage } from './types';
import { marked } from 'marked';

export const URL_SEARCH_PARAM_NAME = 'tag';

export const contentLoaders = import.meta.glob<string>('$lib/content/**/en.md', { as: 'raw' });

export async function getContentBySlug(slug: string): Promise<ContentPage | undefined> {
	const locale = getLocale();
	const fallbackLocale = baseLocale;

	const entries = Object.entries(contentLoaders).filter(([filePath]) => {
		const splitted = filePath.split('/').slice(0, -1).join('/');
		return splitted.endsWith(slug);
	});

	const entry =
		entries.find(([p]) => p.endsWith(`${locale}.md`)) ??
		entries.find(([p]) => p.endsWith(`${fallbackLocale}.md`));

	if (!entry) return undefined;

	const [, loader] = entry;
	const raw = await loader();
	const { attributes, body } = fm(raw);

	const parsed = pageFrontMatterSchema.safeParse(attributes);
	if (!parsed.success) return undefined;

	return {
		attributes: parsed.data,
		body: body ? marked(body, { async: false }) : '',
		slug
	};
}
