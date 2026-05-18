// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { userOrganization } from '$lib/app-state';

import type { OrganizationsResponse } from '@/pocketbase/types';

import { pb } from '@/pocketbase';

import { fetchAvailableForOrganization, type MobileRunnerListItem } from './utils';

//

class Store {
	constructor(private currentOrganization: () => OrganizationsResponse | undefined) {}

	#runners = $state<MobileRunnerListItem[]>([]);
	#generation = 0;
	#dispose: (() => void) | undefined;

	read() {
		return this.#runners;
	}

	init() {
		if (this.#dispose) return;

		this.#dispose = $effect.root(() => {
			$effect(() => {
				const generation = ++this.#generation;
				const organizationId = this.currentOrganization()?.id;

				if (!organizationId) {
					this.#runners = [];
					return;
				}

				this.refresh(generation);

				let unsubscribe: (() => Promise<void>) | undefined;
				let cancelled = false;

				void pb
					.collection('mobile_runners')
					.subscribe('*', () => {
						this.refresh(generation);
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
	}

	private refresh(generation: number) {
		fetchAvailableForOrganization().match({
			Rejected: (reason) => {
				console.error(reason);
			},
			Resolved: (runners) => {
				if (generation !== this.#generation) return;
				this.#runners = runners;
			}
		});
	}
}

export const store = new Store(() => userOrganization.current);
