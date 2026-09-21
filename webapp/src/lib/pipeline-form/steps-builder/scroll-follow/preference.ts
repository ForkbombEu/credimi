// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { browser } from '$app/environment';

import { createStorageHandlers } from '@/utils/storage';

const STORAGE_KEY = 'pipeline_composer_scroll_follow';

type Stored = { enabled: boolean };

const storage = createStorageHandlers<Stored>(
	STORAGE_KEY,
	new Proxy({} as Storage, {
		get(_target, prop, receiver) {
			const target = globalThis.localStorage;
			if (target == null) return undefined;
			const value = Reflect.get(target, prop, receiver);
			return typeof value === 'function' ? value.bind(target) : value;
		}
	})
);

export function readScrollFollowEnabled(): boolean {
	if (!browser) return false;
	try {
		return storage.get()?.enabled === true;
	} catch {
		return false;
	}
}

export function writeScrollFollowEnabled(enabled: boolean): void {
	if (!browser) return;
	try {
		storage.set({ enabled });
	} catch (error) {
		console.error('Failed to persist scroll follow preference:', error);
	}
}
