// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { userOrganization } from '$lib/app-state';

import { pb } from '@/pocketbase';

import type { RunnerRecord } from './types';

import { onRefreshFailure, onRefreshSuccess } from './catalog-state';
import { fetchRecords } from './query';
import { filterRunners } from './search';

//

export const LIVE_REFRESH_MS = 30_000;

let runners = $state<RunnerRecord[]>([]);
let ready = $state(false);
let generation = 0;
let inFlight: Promise<void> | undefined;
let rootDispose: (() => void) | undefined;
let subscribedOrganizationId: string | undefined;

function snapshot() {
	return { ready, runners };
}

function applySuccess(next: RunnerRecord[]) {
	const nextSnapshot = onRefreshSuccess(snapshot(), next);
	ready = nextSnapshot.ready;
	runners = nextSnapshot.runners;
}

function applyFailure() {
	const nextSnapshot = onRefreshFailure(snapshot());
	ready = nextSnapshot.ready;
	runners = nextSnapshot.runners;
}

export function read() {
	return runners;
}

export function isReady() {
	return ready;
}

export function search(text: string) {
	return filterRunners(read(), text);
}

export function findByPath(path: string) {
	return read().find((runner) => runner.path === path);
}

export function init() {
	if (rootDispose) return;

	rootDispose = $effect.root(() => {
		$effect(() => {
			const organizationId = userOrganization.current?.id;

			if (organizationId === subscribedOrganizationId) {
				return;
			}

			subscribedOrganizationId = organizationId;
			const gen = ++generation;

			if (!organizationId) {
				ready = false;
				runners = [];
				return;
			}

			ready = false;
			runners = [];

			void refresh(gen);

			let unsubscribe: (() => Promise<void>) | undefined;
			let cancelled = false;

			void pb
				.collection('mobile_runners')
				.subscribe('*', () => {
					void refresh(gen);
				})
				.then((unsub) => {
					if (cancelled) {
						void unsub();
						return;
					}
					unsubscribe = unsub;
				});

			return () => {
				cancelled = true;
				void unsubscribe?.();
			};
		});
	});
}

export function dispose() {
	rootDispose?.();
	rootDispose = undefined;
	subscribedOrganizationId = undefined;
	generation += 1;
	runners = [];
	ready = false;
	inFlight = undefined;
}

export function refresh(gen?: number): Promise<void> {
	if (inFlight) return inFlight;

	const activeGeneration = gen ?? generation;
	inFlight = fetchRecords()
		.match({
			Rejected: (reason) => {
				console.error(reason);
				if (activeGeneration === generation) {
					applyFailure();
				}
			},
			Resolved: (next) => {
				if (activeGeneration === generation) {
					applySuccess(next);
				}
			}
		})
		.finally(() => {
			inFlight = undefined;
		});

	return inFlight;
}

export function startLiveRefresh(ms = LIVE_REFRESH_MS) {
	void refresh();
	const intervalId = setInterval(() => {
		void refresh();
	}, ms);

	return () => clearInterval(intervalId);
}
