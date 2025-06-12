// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
import { getLocale, baseLocale } from '@/i18n';
import { z } from 'zod';

export const contentLoaders = import.meta.glob<string>('$lib/content/**/en.md', { as: 'raw' });

export const pageFrontMatterSchema = z.object({
	date: z.coerce.date(),
	updatedOn: z.coerce.date(),
	title: z.string(),
	description: z.string().optional(),
	tags: z.array(z.string())
});
export type PageFrontMatter = z.infer<typeof pageFrontMatterSchema>;
export type PageWithBody = PageFrontMatter & {
	body: string;
	slug: string;
};

export async function getContentBySlug(slug: string): Promise<string | undefined> {
	const locale = getLocale();
	const fallbackLocale = baseLocale;

	// use full path: this key must match key in contentLoaders
	let key = `/src/lib/content/pages/${slug}/${locale}.md`;
	let loader = contentLoaders[key as keyof typeof contentLoaders];

	if (!loader) {
		key = `/src/lib/content/pages/${slug}/${fallbackLocale}.md`;
		loader = contentLoaders[key as keyof typeof contentLoaders];
	}

	if (!loader) return undefined;
	return await loader();
}
