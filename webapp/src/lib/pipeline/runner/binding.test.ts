// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getPath } from '$lib/utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { MobileRunnersResponse, PipelinesResponse } from '@/pocketbase/types';

import * as Runner from './binding';

function pipeline(id: string, yaml: string): PipelinesResponse {
	return { id, yaml } as PipelinesResponse;
}

function runnerRecord(path: string): MobileRunnersResponse {
	const segments = path.split('/');
	return {
		id: 'runner-1',
		__canonified_path__: path,
		canonified_name: segments.at(-1) ?? 'runner',
		collectionId: 'mobile_runners',
		collectionName: 'mobile_runners'
	} as unknown as MobileRunnersResponse;
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
		expect(Runner.getExecutionRunnerPath(pipeline('p1', NO_MOBILE_YAML))).toBeUndefined();
	});

	it('returns undefined for global pipeline with no stored runner', () => {
		expect(Runner.getExecutionRunnerPath(pipeline('p2', GLOBAL_MOBILE_YAML))).toBeUndefined();
	});

	it('returns stored path for global pipeline with selected runner', () => {
		const p = pipeline('p3', GLOBAL_MOBILE_YAML);
		const r = runnerRecord('org-a/selected-runner');
		Runner.set(p, r);
		expect(Runner.getExecutionRunnerPath(p)).toBe(getPath(r));
	});

	it('returns runner_id from first mobile-automation step for specific pipeline', () => {
		expect(Runner.getExecutionRunnerPath(pipeline('p4', SPECIFIC_MOBILE_YAML))).toBe(
			'org-a/my-runner'
		);
	});
});
