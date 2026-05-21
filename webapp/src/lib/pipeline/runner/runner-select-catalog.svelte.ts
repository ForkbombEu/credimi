// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Pipeline } from '$lib';
import { Search } from '$lib/pipeline-form/steps/_partials/search.svelte.js';

import type { RunnerRecord } from './types';

//

export type RunnerSelectPresentation = 'minimal' | 'picker' | 'run';

type BindOptions = {
	search?: Search;
};

export function bindRunnerCatalogSearch(options: BindOptions = {}) {
	let foundRunners = $state<RunnerRecord[]>([]);
	const catalogLoading = $derived(!Pipeline.Runner.Catalog.isReady());

	const runnerSearch =
		options.search ??
		new Search({
			onSearch: () => {
				syncFoundRunners();
			}
		});

	function syncFoundRunners() {
		foundRunners = Pipeline.Runner.Catalog.search(runnerSearch.text);
	}

	$effect(() => {
		void Pipeline.Runner.Catalog.isReady();
		void Pipeline.Runner.Catalog.read();
		syncFoundRunners();
	});

	return {
		get foundRunners() {
			return foundRunners;
		},
		get catalogLoading() {
			return catalogLoading;
		},
		runnerSearch
	};
}
