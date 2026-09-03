<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Pipeline } from '$lib';
	import { ItemCard } from '$lib/pipeline-form/steps/_partials/index.js';

	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';

	import type { DeviceSelectPresentation } from './device-select-catalog.svelte.js';
	import type { DeviceRecord } from './types';

	type Props = {
		item: DeviceRecord;
		presentation: DeviceSelectPresentation;
		selectedDevice?: string;
		onSelect?: (device: DeviceRecord) => void;
	};

	let { item, presentation, selectedDevice, onSelect }: Props = $props();

	const isSelected = $derived(selectedDevice === item.path);
	const online = $derived(
		presentation === 'run' && Pipeline.Device.Catalog.isReady() ? item.isOnline : undefined
	);
	const isOffline = $derived(presentation === 'run' && online === false);
</script>

<ItemCard
	title={item.name}
	onClick={isOffline
		? undefined
		: (e) => {
				e.preventDefault();
				onSelect?.(item);
			}}
	class={[
		'hover:cursor-pointer',
		presentation !== 'minimal' && isSelected && 'border-blue-500 bg-blue-50!',
		isOffline && 'bg-slate-100! opacity-50 hover:cursor-not-allowed'
	]}
	tooltip={isOffline ? 'Offline devices cannot be selected' : undefined}
	hideArrow
>
	{#snippet afterContent()}
		{#if item.runnerName}
			<div class="text-xs text-muted-foreground">{item.runnerName}</div>
		{/if}
		{#if item.description}
			<div class="text-xs text-balance text-muted-foreground">{item.description}</div>
		{/if}
	{/snippet}

	{#snippet titleRight()}
		{#if presentation !== 'minimal' && isSelected}
			<Badge
				variant="secondary"
				class="-translate-y-px border border-blue-600 bg-blue-100 px-1 py-0 text-[10px] text-blue-600"
			>
				{m.selected()}
			</Badge>
		{/if}
	{/snippet}

	{#snippet right()}
		<div class="flex items-center gap-2">
			{#if !item.isPublished}
				<Badge variant="secondary">
					{m.private()}
				</Badge>
			{/if}

			{#if presentation === 'run'}
				<span
					class={[
						'size-2 shrink-0 rounded-full border',
						online === true && 'border-emerald-500 bg-emerald-100',
						online === false && 'border-red-500 bg-red-100',
						online === undefined && 'border-muted-foreground/40 bg-muted-foreground/40'
					]}
					title={online === undefined
						? 'Checking device status'
						: online === true
							? 'Online'
							: 'Offline'}
				></span>
			{/if}
		</div>
	{/snippet}
</ItemCard>
