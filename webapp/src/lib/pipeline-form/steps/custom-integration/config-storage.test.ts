// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { CustomChecksResponse } from '@/pocketbase/types';

vi.mock('$app/environment', () => ({ browser: true }));

vi.mock('$lib/utils', () => ({
	getPath: vi.fn((record: { canonified_name?: string }, trim?: boolean) =>
		trim ? (record.canonified_name ?? '') : (record.canonified_name ?? '')
	)
}));

import { getStoredConfig, resolveInitialConfig, setStoredConfig } from './config-storage.js';

const STORAGE_KEY = 'pipeline_custom_integration_configs';

const integration = {
	id: 'ci1',
	canonified_name: 'org/my-integration',
	input_json_sample: { apiKey: 'sample-should-not-be-used' }
} as CustomChecksResponse;

describe('config-storage', () => {
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

	it('getStoredConfig returns undefined when empty', () => {
		expect(getStoredConfig('org/my-integration')).toBeUndefined();
	});

	it('setStoredConfig and getStoredConfig round-trip', () => {
		setStoredConfig('org/my-integration', { apiKey: 'saved' });
		expect(getStoredConfig('org/my-integration')).toEqual({ apiKey: 'saved' });
	});

	it('stores multiple integrations independently', () => {
		setStoredConfig('org/a', { x: 1 });
		setStoredConfig('org/b', { y: 2 });
		expect(getStoredConfig('org/a')).toEqual({ x: 1 });
		expect(getStoredConfig('org/b')).toEqual({ y: 2 });
	});

	it('setStoredConfig preserves other integrations', () => {
		setStoredConfig('org/a', { x: 1 });
		setStoredConfig('org/b', { y: 2 });
		expect(getStoredConfig('org/a')).toEqual({ x: 1 });
		const raw = localStorage.getItem(STORAGE_KEY);
		expect(raw).toBeTruthy();
		expect(JSON.parse(raw!)).toEqual({
			'org/a': { x: 1 },
			'org/b': { y: 2 }
		});
	});

	it('resolveInitialConfig prefers explicit config over localStorage', () => {
		setStoredConfig('org/my-integration', { apiKey: 'from-storage' });
		expect(resolveInitialConfig(integration, { apiKey: 'from-yaml' })).toEqual({
			apiKey: 'from-yaml'
		});
	});

	it('resolveInitialConfig uses localStorage when no explicit config', () => {
		setStoredConfig('org/my-integration', { apiKey: 'from-storage' });
		expect(resolveInitialConfig(integration)).toEqual({ apiKey: 'from-storage' });
	});

	it('resolveInitialConfig returns undefined when no explicit config and no localStorage', () => {
		expect(resolveInitialConfig(integration)).toBeUndefined();
	});

	it('getStoredConfig returns undefined when localStorage throws', () => {
		vi.stubGlobal('localStorage', {
			getItem: () => {
				throw new Error('quota exceeded');
			},
			setItem: vi.fn(),
			removeItem: vi.fn(),
			clear: vi.fn()
		});
		const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
		expect(getStoredConfig('org/my-integration')).toBeUndefined();
		errorSpy.mockRestore();
	});
});
