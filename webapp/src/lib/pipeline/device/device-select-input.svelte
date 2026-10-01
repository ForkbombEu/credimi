<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { SearchInput } from '$lib/pipeline-form/steps/_partials/index.js';
	import { fly } from 'svelte/transition';

	import Label from '@/components/ui/label/label.svelte';

	import type { DeviceRecord } from './types';

	import { bindDeviceCatalogSearch } from './device-select-catalog.svelte.js';
	import DeviceSelectList from './device-select-list.svelte';

	//

	type Presentation = 'picker' | 'run';

	type Props = {
		presentation?: Presentation;
		onSelect?: (device: DeviceRecord) => void;
		selectedDevice?: string;
		name?: string;
		required?: boolean;
		/** Bound-height ScrollArea for the results list (dialogs / dense forms). */
		scrollable?: boolean;
	};

	let {
		presentation = 'picker',
		onSelect,
		selectedDevice,
		name,
		required = false,
		scrollable = false
	}: Props = $props();

	const deviceCatalog = bindDeviceCatalogSearch();
</script>

{#if deviceCatalog.catalogLoading}
	<DeviceSelectList
		{presentation}
		foundDevices={deviceCatalog.foundDevices}
		catalogLoading={deviceCatalog.catalogLoading}
		{onSelect}
		{selectedDevice}
	/>
{:else}
	<div class="space-y-3" transition:fly>
		<div class="space-y-2">
			<Label for={name}>
				Device
				{#if required}
					<span class="font-bold text-destructive">*</span>
				{/if}
			</Label>
			<SearchInput search={deviceCatalog.deviceSearch} {name} />
		</div>

		<DeviceSelectList
			{presentation}
			foundDevices={deviceCatalog.foundDevices}
			catalogLoading={deviceCatalog.catalogLoading}
			{onSelect}
			{selectedDevice}
			{scrollable}
			listContainerClass={scrollable ? 'h-[min(16rem,calc(100dvh-20rem))]' : undefined}
		/>
	</div>
{/if}
