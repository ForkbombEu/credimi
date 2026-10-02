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
      url: https://example.com
      method: GET`);
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

	it('orders with keys as scalars, then arrays, then objects by ascending key count', () => {
		const step = {
			with: {
				body: { hello: 'world', nested: true, extra: 1 },
				method: 'POST',
				tags: ['a', 'b'],
				url: 'https://example.com',
				headers: { Accept: 'application/json' }
			},
			id: 'http-request-0001',
			use: 'http-request'
		} as PipelineStep;

		const yaml = createPipelineYaml('ordered', [step], {});

		expect(yaml).toContain(`with:
      url: https://example.com
      method: POST
      tags:
        - a
        - b
      headers:
        Accept: application/json
      body:
        hello: world
        nested: true
        extra: 1`);
	});

	it('puts http-request with keys url, method, expected_status first', () => {
		const step = {
			with: {
				expected_status: 200,
				headers: { Accept: 'application/json' },
				method: 'GET',
				url: 'https://example.com'
			},
			id: 'http-request-0001',
			use: 'http-request'
		} as PipelineStep;

		const yaml = createPipelineYaml('ordered', [step], {});

		expect(yaml).toContain(`with:
      url: https://example.com
      method: GET
      expected_status: 200
      headers:
        Accept: application/json`);
	});

	it('orders runtime keys as scalars, then arrays, then objects by ascending key count', () => {
		const yaml = createPipelineYaml('ordered', [], {
			temporal: {
				activity_options: {
					retry_policy: { maximum_attempts: 3 },
					start_to_close_timeout: '20m'
				}
			},
			disable_android_play_store: false,
			global_timeout: 60
		} as never);

		expect(yaml).toContain(`runtime:
  disable_android_play_store: false
  global_timeout: 60
  temporal:
    activity_options:
      start_to_close_timeout: 20m
      retry_policy:
        maximum_attempts: 3`);
	});
});
