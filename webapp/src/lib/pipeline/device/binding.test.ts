// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { PipelinesResponse } from '@/pocketbase/types';

import type { DeviceRecord as Record } from './types';

import * as Binding from './binding.js';

function pipeline(id: string, yaml: string): PipelinesResponse {
	return { id, yaml } as PipelinesResponse;
}

function deviceRecord(path: string): Record {
	return {
		isOnline: true,
		isOwned: true,
		isPublished: true,
		name: path.split('/').at(-1) ?? 'device',
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
      device_id: org-a/host-a/device-a`;

describe('getExecutionDevicePath', () => {
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
		expect(Binding.getExecutionDevicePath(pipeline('p1', NO_MOBILE_YAML))).toBeUndefined();
	});

	it('returns undefined for global pipeline with no stored device', () => {
		expect(Binding.getExecutionDevicePath(pipeline('p2', GLOBAL_MOBILE_YAML))).toBeUndefined();
	});

	it('returns stored path for global pipeline with selected device', () => {
		const p = pipeline('p3', GLOBAL_MOBILE_YAML);
		const r = deviceRecord('org-a/selected-device');
		Binding.set(p, r);
		expect(Binding.getExecutionDevicePath(p)).toBe(r.path);
	});

	it('returns device_id from first mobile-automation step for specific pipeline', () => {
		expect(Binding.getExecutionDevicePath(pipeline('p4', SPECIFIC_MOBILE_YAML))).toBe(
			'org-a/host-a/device-a'
		);
	});
});
