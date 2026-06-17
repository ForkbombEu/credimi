// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { validateYaml } from './validate-yaml';

const VALID_YAML = `name: test-pipeline

steps:
  - use: debug
`;

describe('validateYaml', () => {
	it('returns ok with value for valid pipeline yaml', () => {
		const result = validateYaml(VALID_YAML);
		expect(result).toEqual({ ok: true, value: VALID_YAML });
	});

	it('returns error for malformed yaml', () => {
		const result = validateYaml('name: [\n');
		expect(result.ok).toBe(false);
		if (!result.ok) expect(result.message).toMatch(/Invalid YAML/i);
	});

	it('returns error for schema violation', () => {
		const result = validateYaml('name: test\nsteps: {}\n');
		expect(result.ok).toBe(false);
		if (!result.ok) expect(result.message.length).toBeGreaterThan(0);
	});
});
