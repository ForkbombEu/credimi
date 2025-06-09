// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { PageServerLoad } from './$types';
import fs from 'fs';
import path from 'path';
import fm from 'front-matter';
import { marked } from 'marked';
import { getLocale, baseLocale } from '@/i18n/index.js';

interface Page {
	slug: string;
	title: string;
	description: string;
	date: string;
	tags: string[];
	updatedOn: string;
	rawBody: string;
	html: string;
}

function getAllMdFilesForLocale(rootdir: string, locale: string, baseLocale: string): string[] {
	const entries = fs.readdirSync(rootdir, { withFileTypes: true });
	const mdFiles: string[] = [];
	const mdNames = entries
		.filter((e) => e.isFile() && e.name.toLowerCase().endsWith('.md'))
		.map((e) => e.name);

	if (mdNames.length > 0) {
		const desired = `${locale}.md`;
		const fallback = `${baseLocale}.md`;

		if (mdNames.includes(desired)) {
			mdFiles.push(path.join(rootdir, desired));
		} else if (mdNames.includes(fallback)) {
			mdFiles.push(path.join(rootdir, fallback));
		}
		return mdFiles;
	}

	for (const entry of entries) {
		if (entry.isDirectory()) {
			const subdir = path.join(rootdir, entry.name);
			mdFiles.push(...getAllMdFilesForLocale(subdir, locale, baseLocale));
		}
	}

	return mdFiles;
}

export const load: PageServerLoad<{
	pages: Page[];
	initialTag: string;
}> = async ({ url }) => {
	const pagesDir = path.resolve('static', 'pages');
	let files: string[] = [];
	try {
		files = getAllMdFilesForLocale(pagesDir, getLocale(), baseLocale);
	} catch (e) {
		console.error(
			`Failed to scan static/pages recursively: ${e instanceof Error ? e.message : e}`
		);
	}

	const pages: Page[] = [];

	for (const fullPath of files) {
		const fileContents = fs.readFileSync(fullPath, 'utf-8');
		const parsed = fm<{
			title: string;
			description: string;
			date: string;
			tags: string[];
			updatedOn: string;
		}>(fileContents);

		const { attributes, body } = parsed;
		const { title, description, date, tags, updatedOn } = attributes;
		const html = await marked(body);
		const relative = path.relative(pagesDir, fullPath);
		const slug = path.dirname(relative);

		pages.push({
			slug,
			title,
			description,
			date,
			tags: Array.isArray(tags) ? tags : [],
			updatedOn,
			rawBody: body,
			html
		});
	}

	pages.sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime());

	const tagParam = url.searchParams.get('tag') ?? '';

	return {
		pages,
		initialTag: tagParam
	};
};
