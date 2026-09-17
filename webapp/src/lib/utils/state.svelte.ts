// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { resource, type ResourceReturn } from 'runed';
import { onMount } from 'svelte';

import { activeSheet } from './sheet-state.svelte.js';

//

type InitialValueType<T> = (() => T) | undefined;

type PolledResourceOptions<T, InitialValue extends InitialValueType<T>> = {
	intervalMs: number;
	initialValue: InitialValue;
	/** Reactive getters; when any change, Runed `resource` refetches immediately. */
	deps?: Array<() => unknown>;
};

/** Narrow wrapper: Runed's overloads use Awaited<> in a way that fights unconstrained T. */
function createResource<T>(
	deps: Array<() => unknown>,
	fn: () => Promise<T>,
	options: { lazy: true; initialValue?: T }
): ResourceReturn<T> {
	return (
		resource as (
			sources: Array<() => unknown>,
			fetcher: () => Promise<T>,
			opts: { lazy: true; initialValue?: T }
		) => ResourceReturn<T>
	)(deps, fn, options);
}

export class PolledResource<T, InitialValue extends InitialValueType<T>> {
	#paused = $state(false);
	#inner: ResourceReturn<T>;

	constructor(
		private readonly fn: () => Promise<T>,
		options: Partial<PolledResourceOptions<T, InitialValue>> = {}
	) {
		const { intervalMs = 1000, initialValue = undefined, deps = [] } = options;

		this.#inner = initialValue
			? createResource(deps, () => this.fn(), {
					lazy: true,
					initialValue: initialValue()
				})
			: createResource(deps, () => this.fn(), { lazy: true });

		// Re-apply load data when Kit navigations update the thunk (same as prior $effect).
		if (initialValue) {
			$effect(() => {
				this.#inner.mutate(initialValue());
			});
		}

		onMount(() => {
			if (!initialValue) this.fetch();

			const interval = setInterval(() => {
				this.fetch();
			}, intervalMs);

			return () => {
				clearInterval(interval);
			};
		});
	}

	pause() {
		this.#paused = true;
	}

	resume() {
		this.#paused = false;
	}

	get paused() {
		return this.#paused;
	}

	fetch() {
		if (this.#paused || activeSheet.count > 0) return;
		return this.#inner.refetch();
	}

	get current(): InitialValue extends () => T ? T : T | undefined {
		return this.#inner.current as InitialValue extends () => T ? T : T | undefined;
	}

	get error() {
		return this.#inner.error;
	}

	get loading() {
		return this.#inner.loading;
	}

	/** True only while the first fetch is in flight and no value is available yet. */
	get initialLoading() {
		return this.#inner.loading && this.#inner.current == null;
	}
}
