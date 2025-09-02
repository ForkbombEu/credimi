// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { browser } from '$app/environment';

import { pb } from '@/pocketbase';

// Reference: https://github.com/pocketbase/js-sdk/issues/85

export async function verifyUser(fetchFn = fetch): Promise<boolean> {
	if (!browser) throw new Error('verifyUser() must be called from the browser');
	if (!document || !document.cookie) return false;

	pb.authStore.loadFromCookie(document.cookie);
	try {
		await pb.collection('users').authRefresh({ fetch: fetchFn });
	} catch {
		pb.authStore.clear();
		return false;
	}
	return true;
}
