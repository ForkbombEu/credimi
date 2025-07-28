// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import fg from 'fast-glob';
import fs from 'node:fs';
import path from 'node:path';
import fm from 'front-matter';
import { GENERATED, logCodegenResult } from '@/utils/codegen';
import { pageFrontMatterSchema } from './types';

const STRIP_PATH_MARKER = 'pages/';

const tagMap: Record<string, string[]> = {};
const base = import.meta.dirname;
const files = await fg(path.join(base, '**/en.md'));

function stripPagesAndFile(fullPath: string): string {
	const idx = fullPath.indexOf(STRIP_PATH_MARKER);
	if (idx < 0) return '';

	const afterMarker = fullPath.slice(idx + STRIP_PATH_MARKER.length);

	// drop the file
	const parts = afterMarker.split('/');
	if (parts.length <= 1) return '';
	parts.pop();
	return parts.join('/');
}

for (const fullPath of files) {
	const raw = fs.readFileSync(fullPath, 'utf8');
	const parsed = fm(raw);
	const parsedResult = pageFrontMatterSchema.safeParse(parsed.attributes);

	if (!parsedResult.success) {
		console.error(`file ${fullPath} has failed schema validation`);
		continue;
	}
	const tags = parsedResult.data.tags;

	const loaderKey = stripPagesAndFile(fullPath);

	for (const tag of tags) {
		tagMap[tag] ??= [];
		if (loaderKey) tagMap[tag].push(loaderKey);
	}
}

const filePath = path.join(import.meta.dirname, `tags-list.${GENERATED}.json`);
fs.writeFileSync(filePath, JSON.stringify(tagMap, null, 2), 'utf8');
logCodegenResult('tags index JSON', filePath);
