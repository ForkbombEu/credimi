// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
import { baseLocale, getLocale } from '@/i18n/paraglide/runtime';
import fm from 'front-matter';
import { pageFrontMatterSchema, type ContentPage } from './types';
import { marked } from 'marked';

export const contentLoaders = import.meta.glob<string>('./**/en.md', { as: 'raw' });

export async function getContentBySlug(slug: string): Promise<ContentPage | undefined> {
	const locale = getLocale();
	const fallbackLocale = baseLocale;
	const entries = Object.entries(contentLoaders).filter(([filePath]) => {
		const parts = filePath.split('/');
		return parts.length >= 2 && parts[parts.length - 2] === slug;
	});

	const key =
		entries.find(([p]) => p.endsWith(`${locale}.md`))?.[0] ??
		entries.find(([p]) => p.endsWith(`${fallbackLocale}.md`))?.[0];

	if (!key) return undefined;

	const loader = contentLoaders[key as keyof typeof contentLoaders];
	if (!loader) return undefined;

	const raw = await loader();
	const { attributes, body } = fm(raw);

	const parsed = pageFrontMatterSchema.safeParse(attributes);
	if (!parsed.success) return undefined;

	return {
		attributes: parsed.data,
		body: body ? marked(body, { async: false }) : "",
		slug
	};
}
