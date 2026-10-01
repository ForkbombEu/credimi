// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

vi.mock('./steps', () => ({
	getConfigByTypeOrThrow: (use: string) => ({
		use,
		makeId: () => use,
		linkProcedure: undefined
	})
}));

import type { PipelineStep } from '$lib/pipeline/types';

import { createPipelineYaml } from './functions.js';

describe('createPipelineYaml step key order', () => {
	it('emits use, id, continue_on_error, activity_options, with in that order', () => {
		const step = {
			with: {
				method: 'GET',
				url: 'https://example.com'
			},
			activity_options: {
				StartToCloseTimeout: '30s'
			},
			continue_on_error: true,
			id: 'http-request-0001',
			use: 'http-request'
		} as PipelineStep;

		const yaml = createPipelineYaml('ordered', [step], {});

		expect(yaml).toContain(`steps:
  - use: http-request
    id: http-request-0001
    continue_on_error: true
    activity_options:
      StartToCloseTimeout: 30s
    with:
      method: GET
      url: https://example.com`);
	});

	it('omits absent optional keys while keeping the relative order of present ones', () => {
		const step = {
			with: {
				recipient: 'a@b.c',
				subject: 's',
				body: 'b'
			},
			id: 'email-0001',
			use: 'email'
		} as PipelineStep;

		const yaml = createPipelineYaml('ordered', [step], {});

		expect(yaml).toContain(`steps:
  - use: email
    id: email-0001
    with:
      recipient: a@b.c
      subject: s
      body: b`);
		expect(yaml).not.toContain('continue_on_error');
		expect(yaml).not.toContain('activity_options');
	});

	it('applies the same key order under finally (without continue_on_error)', () => {
		const step = {
			with: {
				recipient: 'a@b.c',
				subject: 's',
				body: 'b'
			},
			continue_on_error: true,
			id: 'email-0001',
			use: 'email'
		} as PipelineStep;

		const yaml = createPipelineYaml('ordered', [], {}, [
			{ step: [step, {}], condition: 'always' }
		]);

		expect(yaml).toContain(`finally:
  always:
    - use: email
      id: email-0001
      with:
        recipient: a@b.c
        subject: s
        body: b`);
		expect(yaml).not.toMatch(/finally:[\s\S]*continue_on_error/);
	});
});
