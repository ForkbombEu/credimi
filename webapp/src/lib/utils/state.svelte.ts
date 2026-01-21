// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { onMount } from 'svelte';

//

type PolledResourceOptions<T> = {
	intervalMs: number;
	initialValue: (() => T) | undefined;
};

export class PolledResource<T> {
	constructor(
		private readonly fn: () => Promise<T>,
		options: Partial<PolledResourceOptions<T>> = {}
	) {
		const { intervalMs = 1000, initialValue = undefined } = options;

		$effect(() => {
			this.#current = initialValue?.();
		});

		onMount(() => {
			const interval = setInterval(() => {
				this.fetch();
			}, intervalMs);

			return () => {
				clearInterval(interval);
			};
		});
	}

	#current = $state<T>();
	#error = $state<Error>();
	#loading = $state(false);

	async fetch() {
		this.#loading = true;
		try {
			this.#current = await this.fn();
		} catch (error) {
			this.#error = error as Error;
		} finally {
			this.#loading = false;
		}
	}

	get current() {
		return this.#current;
	}

	get error() {
		return this.#error;
	}

	get loading() {
		return this.#loading;
	}
}
