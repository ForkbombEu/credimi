// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { browser } from '$app/environment';
import { writable } from 'svelte/store';

import { pb } from '@/pocketbase';

import { Features } from './features-list.generated';

//

export const featureFlags = writable<FeatureFlags>();

export type FeatureKeys = keyof typeof Features;
export type FeatureFlags = Record<FeatureKeys, boolean>;

export async function loadFeatureFlags(fetchFn = fetch): Promise<FeatureFlags> {
	const flags: Partial<FeatureFlags> = {};

	const list = await pb.collection('features').getFullList({ requestKey: null, fetch: fetchFn });

	for (const [key, name] of Object.entries(Features)) {
		const feature = list.find((f) => f.name === name);
		flags[key as FeatureKeys] = feature?.active ?? false;
	}

	if (browser) featureFlags.set(flags as FeatureFlags);
	return flags as FeatureFlags;
}

//

export * from './features-list.generated';
