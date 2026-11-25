// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import fg from 'fast-glob';
import fm from 'front-matter';
import fs from 'node:fs';
import path from 'node:path';

import { GENERATED, logCodegenResult } from '@/utils/codegen';

import { pageFrontMatterSchema } from './types';

const tagMap: Record<string, string[]> = {};
const base = import.meta.dirname;
const pagesBase = path.resolve(base, '../../../static/pages');
const files = await fg('*/**/en.md', { cwd: pagesBase });

for (const file of files) {
	const fullPath = path.join(pagesBase, file);
	const raw = fs.readFileSync(fullPath, 'utf8');
	const parsed = fm(raw);
	const parsedResult = pageFrontMatterSchema.safeParse(parsed.attributes);

	if (!parsedResult.success) {
		console.error(`file ${fullPath} has failed schema validation`);
		continue;
	}
	const tags = parsedResult.data.tags;

	const loaderKey = path.posix.dirname(file);

	for (const tag of tags) {
		tagMap[tag] ??= [];
		tagMap[tag].push(loaderKey);
	}
}

const filePath = path.join(import.meta.dirname, `tags-list.${GENERATED}.json`);
fs.writeFileSync(filePath, JSON.stringify(tagMap, null, 2), 'utf8');
logCodegenResult('tags index JSON', filePath);
