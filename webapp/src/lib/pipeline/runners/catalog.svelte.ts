// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { userOrganization } from '$lib/app-state';

import type { OrganizationsResponse } from '@/pocketbase/types';

import { pb } from '@/pocketbase';

import { listSelector } from './query';
import { filterRunners } from './search';
import type { RunnerRecord } from './types';

//

export const LIVE_REFRESH_MS = 30_000;

class Catalog {
	constructor(private currentOrganization: () => OrganizationsResponse | undefined) {}

	#runners = $state<RunnerRecord[]>([]);
	#ready = $state(false);
	#generation = 0;
	#inFlight: Promise<void> | undefined;
	#dispose: (() => void) | undefined;

	read() {
		return this.#runners;
	}

	isReady() {
		return this.#ready;
	}

	search(text: string) {
		return filterRunners(this.read(), text);
	}

	findByPath(path: string) {
		return this.read().find((runner) => runner.path === path);
	}

	init() {
		if (this.#dispose) return;

		this.#dispose = $effect.root(() => {
			$effect(() => {
				const generation = ++this.#generation;
				const organizationId = this.currentOrganization()?.id;

				this.#ready = false;
				this.#runners = [];

				if (!organizationId) {
					return;
				}

				void this.refresh(generation);

				let unsubscribe: (() => Promise<void>) | undefined;
				let cancelled = false;

				void pb
					.collection('mobile_runners')
					.subscribe('*', () => {
						void this.refresh(generation);
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

	dispose() {
		this.#dispose?.();
		this.#dispose = undefined;
		this.#generation += 1;
		this.#runners = [];
		this.#ready = false;
		this.#inFlight = undefined;
	}

	refresh(generation?: number): Promise<void> {
		if (this.#inFlight) return this.#inFlight;

		const gen = generation ?? this.#generation;
		this.#inFlight = listSelector()
			.match({
				Rejected: (reason) => {
					console.error(reason);
					if (gen === this.#generation) {
						this._applyRefreshFailure();
					}
				},
				Resolved: (runners) => {
					if (gen === this.#generation) {
						this._applyRefreshSuccess(runners);
					}
				}
			})
			.finally(() => {
				this.#inFlight = undefined;
			});

		return this.#inFlight;
	}

	startLiveRefresh(ms = LIVE_REFRESH_MS) {
		void this.refresh();
		const intervalId = setInterval(() => {
			void this.refresh();
		}, ms);

		return () => clearInterval(intervalId);
	}

	_applyRefreshSuccess(runners: RunnerRecord[]) {
		this.#runners = runners;
		this.#ready = true;
	}

	_applyRefreshFailure() {
		if (!this.#ready) {
			this.#runners = [];
		}
	}
}

const catalog = new Catalog(() => userOrganization.current);

export function init() {
	catalog.init();
}

export function dispose() {
	catalog.dispose();
}

export function refresh() {
	return catalog.refresh();
}

export function read() {
	return catalog.read();
}

export function search(text: string) {
	return catalog.search(text);
}

export function findByPath(path: string) {
	return catalog.findByPath(path);
}

export function isReady() {
	return catalog.isReady();
}

export function startLiveRefresh(ms = LIVE_REFRESH_MS) {
	return catalog.startLiveRefresh(ms);
}

export function createCatalogState() {
	return new Catalog(() => undefined);
}

export function applyRefreshSuccess(state: Catalog, runners: RunnerRecord[]) {
	state._applyRefreshSuccess(runners);
}

export function applyRefreshFailure(state: Catalog) {
	state._applyRefreshFailure();
}
