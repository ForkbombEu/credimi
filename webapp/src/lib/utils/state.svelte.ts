// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { onMount } from 'svelte';

//

type InitialValueType<T> = (() => T) | undefined;

type PolledResourceOptions<T, InitialValue extends InitialValueType<T>> = {
	intervalMs: number;
	initialValue: InitialValue;
};

export class PolledResource<T, InitialValue extends InitialValueType<T>> {
	constructor(
		private readonly fn: () => Promise<T>,
		options: Partial<PolledResourceOptions<T, InitialValue>> = {}
	) {
		const { intervalMs = 1000, initialValue = undefined } = options;

		this.#current = initialValue?.();
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

	get current(): InitialValue extends () => T ? T : T | undefined {
		return this.#current as T;
	}

	get error() {
		return this.#error;
	}

	get loading() {
		return this.#loading;
	}
}
