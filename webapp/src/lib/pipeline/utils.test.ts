// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';
import { parse } from 'yaml';

import { serializeYaml } from './utils';

// The backend resolves pipeline YAML with gopkg.in/yaml.v3, whose implicit
// timestamp type matches this schema. Anything it treats as a timestamp here
// would be rewritten to RFC3339 when delivered in an HTTP body.
const resolveLikeBackend = (yaml: string) => parse(yaml, { schema: 'yaml-1.1' });

describe('serializeYaml', () => {
	it('quotes date-like strings so they survive the backend parser', () => {
		const out = serializeYaml({
			steps: [
				{
					id: 's',
					use: 'http-request',
					with: { body: { values: ['2012-03-04'] } }
				}
			]
		});

		const value = resolveLikeBackend(out).steps[0].with.body.values[0];
		expect(value).toBe('2012-03-04');
	});

	it('keeps full timestamp strings verbatim', () => {
		const out = serializeYaml({
			steps: [
				{
					id: 's',
					use: 'http-request',
					with: { body: { values: ['2031-01-01T00:00:00Z'] } }
				}
			]
		});

		const value = resolveLikeBackend(out).steps[0].with.body.values[0];
		expect(value).toBe('2031-01-01T00:00:00Z');
	});
});
