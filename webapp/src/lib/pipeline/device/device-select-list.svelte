<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	import { EmptyState, WithEmptyState } from '$lib/pipeline-form/steps/_partials/index.js';
	import { fly } from 'svelte/transition';

	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import type { DeviceSelectPresentation } from './device-select-catalog.svelte.js';
	import type { DeviceRecord } from './types';

	import DeviceSelectListItem from './device-select-list-item.svelte';

	type Props = {
		presentation: DeviceSelectPresentation;
		foundDevices: DeviceRecord[];
		catalogLoading: boolean;
		onSelect?: (device: DeviceRecord) => void;
		selectedDevice?: string;
		scrollable?: boolean;
		prepend?: Snippet;
		emptyContainerClass?: ClassValue;
		listContainerClass?: ClassValue;
	};

	let {
		presentation,
		foundDevices,
		catalogLoading,
		onSelect,
		selectedDevice,
		scrollable = false,
		prepend,
		emptyContainerClass = 'p-0!',
		listContainerClass
	}: Props = $props();
</script>

{#if catalogLoading}
	<EmptyState containerClass={emptyContainerClass}>
		<Spinner size={16} />
		<T>{m.Loading()}</T>
	</EmptyState>
{:else}
	<div class="space-y-1" transition:fly>
		{@render prepend?.()}

		{#if scrollable}
			<WithEmptyState
				items={foundDevices}
				emptyText={'No devices found'}
				containerClass={listContainerClass}
			>
				{#snippet item({ item })}
					<DeviceSelectListItem {item} {presentation} {selectedDevice} {onSelect} />
				{/snippet}
			</WithEmptyState>
		{:else}
			<div class={['space-y-2', listContainerClass]}>
				{#each foundDevices as item (item.path)}
					<DeviceSelectListItem {item} {presentation} {selectedDevice} {onSelect} />
				{:else}
					<EmptyState text={'No devices found'} containerClass={emptyContainerClass} />
				{/each}
			</div>
		{/if}
	</div>
{/if}
