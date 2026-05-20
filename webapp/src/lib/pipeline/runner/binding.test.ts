// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('./run-now-button.svelte', () => ({ default: {} }));
vi.mock('./runner-select-input.svelte', () => ({ default: {} }));
vi.mock('./runner-select-modal.svelte', () => ({ default: {} }));
vi.mock('../runners/catalog.svelte.js', () => ({
	dispose: () => {},
	findByPath: () => undefined,
	init: () => {},
	isReady: () => false,
	read: () => [],
	refresh: async () => {},
	search: () => [],
	startLiveRefresh: () => () => {}
}));

import type { PipelinesResponse } from '@/pocketbase/types';

import type { RunnerRecord as Record } from '../runners/types';

import { Binding } from './index';

function pipeline(id: string, yaml: string): PipelinesResponse {
	return { id, yaml } as PipelinesResponse;
}

function runnerRecord(path: string): Record {
	return {
		isOnline: true,
		isOwned: true,
		isPublished: true,
		name: path.split('/').at(-1) ?? 'runner',
		path
	};
}

const NO_MOBILE_YAML = `steps:
  - use: http-request
    id: step1`;

const GLOBAL_MOBILE_YAML = `steps:
  - use: mobile-automation
    id: ma1
    with: {}`;

const SPECIFIC_MOBILE_YAML = `steps:
  - use: mobile-automation
    id: ma1
    with:
      runner_id: org-a/my-runner`;

describe('getExecutionRunnerPath', () => {
	beforeEach(() => {
		const store = new Map<string, string>();
		vi.stubGlobal('localStorage', {
			clear: () => store.clear(),
			getItem: (key: string) => store.get(key) ?? null,
			removeItem: (key: string) => {
				store.delete(key);
			},
			setItem: (key: string, value: string) => {
				store.set(key, value);
			}
		});
	});

	it('returns undefined when mobile-automation is not required', () => {
		expect(Binding.getExecutionRunnerPath(pipeline('p1', NO_MOBILE_YAML))).toBeUndefined();
	});

	it('returns undefined for global pipeline with no stored runner', () => {
		expect(Binding.getExecutionRunnerPath(pipeline('p2', GLOBAL_MOBILE_YAML))).toBeUndefined();
	});

	it('returns stored path for global pipeline with selected runner', () => {
		const p = pipeline('p3', GLOBAL_MOBILE_YAML);
		const r = runnerRecord('org-a/selected-runner');
		Binding.set(p, r);
		expect(Binding.getExecutionRunnerPath(p)).toBe(r.path);
	});

	it('returns runner_id from first mobile-automation step for specific pipeline', () => {
		expect(Binding.getExecutionRunnerPath(pipeline('p4', SPECIFIC_MOBILE_YAML))).toBe(
			'org-a/my-runner'
		);
	});
});
