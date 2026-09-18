// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

import { parseFCAFTestId } from './categories';

type ParseCase = {
	name: string;
	test_id: string;
	code: string;
	subgroup: string;
	label: string;
	category_label: string;
};

const casesPath = path.resolve(
	path.dirname(fileURLToPath(import.meta.url)),
	'../../../../pkg/fcaf/taxonomy/testdata/parse_cases.json'
);
const cases = JSON.parse(readFileSync(casesPath, 'utf8')) as ParseCase[];

describe('parseFCAFTestId', () => {
	it.each(cases)('$name', (tc) => {
		const parsed = parseFCAFTestId(tc.test_id || undefined);
		expect(parsed.category.code).toBe(tc.code);
		expect(parsed.category.label).toBe(tc.category_label);
		expect(parsed.key).toBe(tc.subgroup);
		expect(parsed.label).toBe(tc.label);
	});
});
