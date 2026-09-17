// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ExecutionSummary } from '$lib/pipeline/workflows';

import { Pipeline } from '$lib';
import { activeSheet } from '$lib/utils/sheet-state.svelte.js';
import { Effect } from 'effect';
import { onMount } from 'svelte';

//

export type PipelineListExecutionsSection = 'owned' | 'public';

export type PipelineListExecutionsEntry = {
	workflows?: ExecutionSummary[];
	error?: Error;
	/** True only during the first in-flight fetch (drives skeleton). */
	loading: boolean;
	/** True after at least one fetch attempt finished (success or error). */
	hydrated: boolean;
};

const POLL_INTERVAL_MS = 30_000;
const FETCH_CONCURRENCY = 5;
const LIST_LIMIT = 5;

function uniqueIds(ids: string[]): string[] {
	const out: string[] = [];
	for (const id of ids) {
		if (!id || out.includes(id)) continue;
		out.push(id);
	}
	return out;
}

/**
 * Page-level store for `/my/pipelines`: shared 30s tick, per-pipeline history
 * (`limit=5`), Effect concurrency cap, first-load skeleton via `loading`.
 */
export class PipelineListExecutions {
	#sections = $state<Record<PipelineListExecutionsSection, string[]>>({
		owned: [],
		public: []
	});
	#entries = $state<Record<string, PipelineListExecutionsEntry>>({});
	#paused = $state(false);
	#refreshAllInFlight = false;

	constructor() {
		onMount(() => {
			void this.refreshAll();
			const interval = setInterval(() => {
				void this.refreshAll();
			}, POLL_INTERVAL_MS);
			return () => clearInterval(interval);
		});
	}

	getEntry(pipelineId: string): PipelineListExecutionsEntry | undefined {
		return this.#entries[pipelineId];
	}

	setSection(section: PipelineListExecutionsSection, ids: string[]) {
		const next = uniqueIds(ids);
		const prev = this.#sections[section];
		if (prev.join('\0') === next.join('\0')) return;

		this.#sections[section] = next;
		const visible = this.ids;
		const newcomers: string[] = [];

		for (const id of next) {
			if (!this.#entries[id]) {
				newcomers.push(id);
			}
		}

		const pruned: Record<string, PipelineListExecutionsEntry> = {};
		for (const id of visible) {
			pruned[id] = this.#entries[id] ?? { loading: true, hydrated: false };
		}
		this.#entries = pruned;

		if (newcomers.length > 0) {
			void this.#fetchIds(newcomers, { firstLoad: true });
		}
	}

	get ids(): string[] {
		return uniqueIds([...this.#sections.owned, ...this.#sections.public]);
	}

	pause() {
		this.#paused = true;
	}

	resume() {
		this.#paused = false;
	}

	async refreshAll() {
		if (this.#paused || activeSheet.count > 0 || this.#refreshAllInFlight) return;
		const ids = this.ids;
		if (ids.length === 0) return;
		this.#refreshAllInFlight = true;
		try {
			await this.#fetchIds(ids, { firstLoad: false });
		} finally {
			this.#refreshAllInFlight = false;
		}
	}

	async retry(pipelineId: string) {
		if (!pipelineId) return;
		await this.#fetchIds([pipelineId], {
			firstLoad: !this.#entries[pipelineId]?.hydrated
		});
	}

	async #fetchIds(ids: string[], options: { firstLoad: boolean }) {
		const targets = uniqueIds(ids);
		if (targets.length === 0) return;

		await Effect.runPromise(
			Effect.forEach(
				targets,
				(pipelineId) => Effect.promise(() => this.#fetchOne(pipelineId, options.firstLoad)),
				{ concurrency: FETCH_CONCURRENCY }
			)
		);
	}

	async #fetchOne(pipelineId: string, allowSkeleton: boolean) {
		const previous = this.#entries[pipelineId];
		const showSkeleton = allowSkeleton && !previous?.hydrated;

		if (showSkeleton) {
			this.#entries = {
				...this.#entries,
				[pipelineId]: {
					workflows: previous?.workflows,
					error: undefined,
					loading: true,
					hydrated: false
				}
			};
		}

		try {
			const workflows = await Pipeline.Workflows.list(pipelineId, {
				limit: LIST_LIMIT,
				page: 0
			});
			this.#entries = {
				...this.#entries,
				[pipelineId]: {
					workflows,
					error: undefined,
					loading: false,
					hydrated: true
				}
			};
		} catch (error) {
			this.#entries = {
				...this.#entries,
				[pipelineId]: {
					workflows: previous?.workflows,
					error: error as Error,
					loading: false,
					hydrated: true
				}
			};
		}
	}
}
