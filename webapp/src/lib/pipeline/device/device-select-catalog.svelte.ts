// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Pipeline } from '$lib';
import { Search } from '$lib/pipeline-form/steps/_partials/index.js';

import type { DeviceRecord } from './types';

//

export type DeviceSelectPresentation = 'minimal' | 'picker' | 'run';

type BindOptions = {
	search?: Search;
};

export function bindDeviceCatalogSearch(options: BindOptions = {}) {
	let foundDevices = $state<DeviceRecord[]>([]);
	const catalogLoading = $derived(!Pipeline.Device.Catalog.isReady());

	const deviceSearch =
		options.search ??
		new Search({
			onSearch: () => {
				syncFoundDevices();
			}
		});

	function syncFoundDevices() {
		foundDevices = Pipeline.Device.Catalog.search(deviceSearch.text);
	}

	$effect(() => {
		void Pipeline.Device.Catalog.isReady();
		void Pipeline.Device.Catalog.read();
		syncFoundDevices();
	});

	return {
		get foundDevices() {
			return foundDevices;
		},
		get catalogLoading() {
			return catalogLoading;
		},
		deviceSearch
	};
}
