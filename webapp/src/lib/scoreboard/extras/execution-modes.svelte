<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ClockIcon, CogIcon, HandIcon } from '@lucide/svelte';

	import type { IconComponent } from '@/components/types';

	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { m } from '@/i18n';

	import type { ExecutionStats } from './from-scoreboard-row';

	//

	type ExecutionModeCount = {
		icon: IconComponent;
		count: number;
		label: string;
	};

	type Props = {
		stats: ExecutionStats;
		/** 'inline' renders counts with separators; 'list' renders one labeled row per mode */
		variant?: 'inline' | 'list';
		tag?: 'p' | 'span';
		class?: string;
	};

	let { stats, variant = 'inline', tag = 'p', class: className }: Props = $props();

	const executionTypes: ExecutionModeCount[] = $derived([
		{ icon: HandIcon, count: stats.manual, label: m.Executed_manually() },
		{ icon: ClockIcon, count: stats.scheduled, label: m.Executed_via_scheduling() },
		{ icon: CogIcon, count: stats.ci, label: m.Executed_via_ci() }
	]);
</script>

{#if variant === 'inline'}
	<svelte:element this={tag} class={className}>
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
	</svelte:element>
{:else}
	<ul class={['flex flex-col gap-1', className]}>
		{#each executionTypes as executionType (executionType.label)}
			<Tooltip>
				<li class="flex items-center gap-2 text-sm">
					<executionType.icon class="size-4 shrink-0 text-muted-foreground" />
					<span class="font-semibold tabular-nums">{executionType.count}</span>
					<span>{executionType.label}</span>
				</li>
				{#snippet content()}
					<p>{executionType.label}</p>
				{/snippet}
			</Tooltip>
		{/each}
	</ul>
{/if}
