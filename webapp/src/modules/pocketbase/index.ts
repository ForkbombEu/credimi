// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import PocketBase, { ClientResponseError } from 'pocketbase';
import { writable } from 'svelte/store';

import type { TypedPocketBase, UsersResponse } from '@/pocketbase/types';

//

export const pb = new PocketBase() as TypedPocketBase;
// On the server this one client serves every concurrent request, and auto-cancellation would let
// one visitor's request abort another's identical one.
if (typeof window === 'undefined') pb.autoCancellation(false);

export const currentUser = writable(pb.authStore.model as AuthStoreModel);
export type AuthStoreModel = UsersResponse | null;

/** `.catch` handler for loads: a record PocketBase does not find is the page's 404, not a 500. */
export function notFoundAs404(err: unknown): never {
	if (err instanceof ClientResponseError && err.status === 404) error(404);
	throw err;
}
