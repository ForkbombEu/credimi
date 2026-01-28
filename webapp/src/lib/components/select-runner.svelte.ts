// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { userOrganization } from '$lib/app-state/index.svelte.js';

import { pb } from '@/pocketbase/index.js';
import type { MobileRunnersResponse } from '@/pocketbase/types';

import { Search } from '$lib/pipeline-form/steps/_partials/search.svelte.js';

//

export interface SelectRunnerData {
	runner: MobileRunnersResponse;
}

export class SelectRunnerForm {
	constructor(onSubmit?: (runner: MobileRunnersResponse) => void) {
		if (onSubmit) {
			this.onSubmit = onSubmit;
		}
		// Trigger initial search
		this.searchRunner('');
	}

	selectedRunner = $state<MobileRunnersResponse | undefined>(undefined);
	foundRunners = $state<MobileRunnersResponse[]>([]);

	runnerSearch = new Search({
		onSearch: (text) => {
			this.searchRunner(text);
		}
	});

	async searchRunner(text: string) {
		const filter = pb.filter(
			[
				['name ~ {:text}', 'canonified_name ~ {:text}'].join(' || '),
				['owner.id = {:currentOrganization}', 'published = true'].join(' || ')
			]
				.map((f) => `(${f})`)
				.join(' && '),
			{
				text: text,
				currentOrganization: userOrganization.current?.id
			}
		);
		this.foundRunners = await pb.collection('mobile_runners').getFullList({
			requestKey: null,
			filter: filter,
			sort: 'created'
		});
	}

	selectRunner(runner: MobileRunnersResponse) {
		this.selectedRunner = runner;
		this.onSubmit?.(runner);
	}

	removeRunner() {
		this.selectedRunner = undefined;
	}

	onSubmit?: (runner: MobileRunnersResponse) => void;
}
