// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { browser } from '$app/environment';
import { NotBrowserError } from './errors';

//

export function createStorageHandlers<T = { [x: string]: boolean }>(key: string, storage: Storage) {
	function browserGuard() {
		if (!browser) throw new NotBrowserError();
	}

	function set(data?: T) {
		browserGuard();
		storage.setItem(key, JSON.stringify(data ?? { [key]: true }));
	}

	function get(): T | null {
		browserGuard();
		const data = storage.getItem(key);
		return data ? JSON.parse(data) : null;
	}

	function isSet(): boolean {
		browserGuard();
		return Boolean(get());
	}

	function remove() {
		browserGuard();
		storage.removeItem(key);
	}

	return {
		set,
		get,
		remove,
		isSet
	};
}

export function createSessionStorageHandlers<T = { [x: string]: boolean }>(key: string) {
	const handlers = createStorageHandlers<T>(key, sessionStorage);

	return {
		start: handlers.set,
		end: handlers.remove,
		getData: handlers.get,
		isActive: handlers.isSet
	};
}
