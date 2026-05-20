// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import RunNowButton from './run-now-button.svelte';
import SelectInput from './runner-select-input.svelte';
import SelectModal from './runner-select-modal.svelte';

import * as binding from './binding';
import * as catalog from '../runners/catalog.svelte.js';

export type { RunnerRecord as Record } from '../runners/types.js';

export const Binding = {
	get: binding.get,
	getExecutionRunnerPath: binding.getExecutionRunnerPath,
	getType: binding.getType,
	isRequired: binding.isRequired,
	set: binding.set
};

export const Catalog = {
	dispose: catalog.dispose,
	findByPath: catalog.findByPath,
	init: catalog.init,
	isReady: catalog.isReady,
	read: catalog.read,
	refresh: catalog.refresh,
	search: catalog.search,
	startLiveRefresh: catalog.startLiveRefresh
};

export { RunNowButton, SelectInput, SelectModal };
