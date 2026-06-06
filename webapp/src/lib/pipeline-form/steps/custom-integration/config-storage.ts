// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getPath } from '$lib/utils';
import type { CustomChecksResponse } from '@/pocketbase/types';
import { createStorageHandlers } from '@/utils/storage';

type CustomIntegrationConfigStore = Record<string, Record<string, unknown>>;

const STORAGE_KEY = 'pipeline_custom_integration_configs';
const storage = createStorageHandlers<CustomIntegrationConfigStore>(
	STORAGE_KEY,
	new Proxy({} as Storage, {
		get(_target, prop, receiver) {
			const target = globalThis.localStorage;
			if (target == null) {
				return undefined;
			}
			const value = Reflect.get(target, prop, receiver);
			return typeof value === 'function' ? value.bind(target) : value;
		}
	})
);

export function getStoredConfig(checkId: string): Record<string, unknown> | undefined {
	try {
		return storage.get()?.[checkId];
	} catch (error) {
		console.error('Failed to get custom integration config:', error);
		return undefined;
	}
}

export function setStoredConfig(checkId: string, config: Record<string, unknown>): void {
	try {
		const current = storage.get() ?? {};
		storage.set({ ...current, [checkId]: config });
	} catch (error) {
		console.error('Failed to set custom integration config:', error);
	}
}

export function resolveInitialConfig(
	integration: CustomChecksResponse,
	explicitConfig?: Record<string, unknown>
): Record<string, unknown> | undefined {
	if (explicitConfig !== undefined) return explicitConfig;
	return getStoredConfig(getPath(integration, true)) ?? undefined;
}
