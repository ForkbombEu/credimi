// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { CHECK_CLIENT_COLUMNS, SUITE_CLIENT_COLUMNS } from './columns';
import { conformanceCheckRecordSchema, conformanceSuiteRecordSchema } from './record';

describe('catalog row schema alignment', () => {
	it('Zod check keys match Go Client=true check columns', () => {
		const shape = conformanceCheckRecordSchema.shape;
		expect(Object.keys(shape).sort()).toEqual([...CHECK_CLIENT_COLUMNS].sort());
	});

	it('Zod suite keys match Go Client=true suite columns', () => {
		const shape = conformanceSuiteRecordSchema.shape;
		expect(Object.keys(shape).sort()).toEqual([...SUITE_CLIENT_COLUMNS].sort());
	});
});
