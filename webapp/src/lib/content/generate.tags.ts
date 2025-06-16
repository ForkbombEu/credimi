// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import fg from 'fast-glob';
import fs from 'node:fs';
import path from 'node:path';
import fm from 'front-matter';
import { GENERATED, logCodegenResult } from '@/utils/codegen';
import { pageFrontMatterSchema } from './types';

const tagMap: Record<string, string[]> = {};
const base = import.meta.dirname;
const files = await fg(path.join(base, '**/en.md'));

for (const fullPath of files) {
	const raw = fs.readFileSync(fullPath, 'utf8');
	const parsed = fm(raw);
	const parsedResult = pageFrontMatterSchema.safeParse(parsed.attributes);
	console.log(parsedResult);
	
	if (!parsedResult.success) {
		console.error(`file ${fullPath} has failed schema validation`);
		continue;
	}
	const tags = parsedResult.data.tags;
	const parts = fullPath.split('/');
	const loaderKey = parts[parts.length - 2];

	for (const tag of tags) {
		tagMap[tag] ??= [];
		tagMap[tag].push(loaderKey);
	}
}

const filePath = path.join(import.meta.dirname, `tags-list.${GENERATED}.json`);
fs.writeFileSync(filePath, JSON.stringify(tagMap, null, 2), 'utf8');
logCodegenResult('tags index JSON', filePath);
