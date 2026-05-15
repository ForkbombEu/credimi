<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ClockIcon, CogIcon, HandIcon } from '@lucide/svelte';

	import type { IconComponent } from '@/components/types';

	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import type { ExecutionStats } from './from-scoreboard-row';

	type Layout = 'inline' | 'stat-box-success' | 'stat-box-modes';

	type Props = {
		stats: ExecutionStats;
		layout: Layout;
		label?: string;
	};

	let { stats, layout, label }: Props = $props();

	type ExecutionModeCount = {
		icon: IconComponent;
		count: number;
		label: string;
	};

	const executionTypes: ExecutionModeCount[] = $derived([
		{ icon: HandIcon, count: stats.manual, label: m.Executed_manually() },
		{ icon: ClockIcon, count: stats.scheduled, label: m.Executed_via_scheduling() },
		{ icon: CogIcon, count: stats.ci, label: m.Executed_via_ci() }
	]);

	const successClass = $derived(stats.percent >= 70 ? 'text-emerald-600' : undefined);
</script>

{#snippet successLine(className?: string)}
	<p class={['font-bold', className, successClass]}>
		{stats.successes}/{stats.total} ({stats.percent}%)
	</p>
{/snippet}

{#snippet modesLine(className?: string)}
	<p class={className}>
		{#each executionTypes as executionType, index (executionType.label)}
			<Tooltip>
				<span>
					{executionType.count}
					<executionType.icon class="-ml-0.5 inline-block size-3 -translate-y-px" />
				</span>
				{#snippet content()}
					<p>
						<executionType.icon class="inline-block size-3 -translate-y-px" />
						{executionType.label}
					</p>
				{/snippet}
			</Tooltip>
			{#if index < executionTypes.length - 1}
				<span class="pr-1 pl-0.5">/</span>
			{/if}
		{/each}
	</p>
{/snippet}

{#if layout === 'inline'}
	<div class="shrink-0 pr-3 text-right">
		{@render successLine('text-sm')}
		{@render modesLine('text-xs text-muted-foreground opacity-80')}
	</div>
{:else if layout === 'stat-box-success'}
	<div class="flex h-20 w-[140px] flex-col items-start justify-between rounded-lg border p-3">
		<T tag="h2" class={['mb-0! pb-0!', successClass]}>
			{stats.successes}/{stats.total} ({stats.percent}%)
		</T>
		<T class="text-sm">{label}</T>
	</div>
{:else}
	<div class="flex h-20 w-[140px] flex-col items-start justify-between rounded-lg border p-3">
		<div class="text-lg leading-tight font-semibold">
			{@render modesLine('text-sm')}
		</div>
		<T class="text-sm">{label}</T>
	</div>
{/if}
