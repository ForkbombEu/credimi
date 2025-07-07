// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { browser } from '$app/environment';
import { NotBrowserError } from './errors';

export function createSessionStorageHandlers<T = { [x: string]: boolean }>(key: string) {
	function browserGuard() {
		if (!browser) throw new NotBrowserError();
	}

	function start(data?: T) {
		browserGuard();
		sessionStorage.setItem(key, JSON.stringify(data ?? { [key]: true }));
	}

	function getData(): T | null {
		browserGuard();
		const data = sessionStorage.getItem(key);
		return data ? JSON.parse(data) : null;
	}

	function isActive(): boolean {
		browserGuard();
		return Boolean(getData());
	}

	function end() {
		browserGuard();
		sessionStorage.removeItem(key);
	}

	return {
		start,
		end,
		getData,
		isActive
	};
}
