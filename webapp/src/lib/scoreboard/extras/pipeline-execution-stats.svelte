<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import type { ExecutionStats } from './from-scoreboard-row';

	import ExecutionModes from './execution-modes.svelte';

	//

	type Layout = 'inline' | 'card-inline' | 'stat-box-success' | 'stat-box-modes';

	type Props = {
		stats: ExecutionStats;
		layout: Layout;
		label?: string;
	};

	let { stats, layout, label }: Props = $props();

	const successClass = $derived(stats.percent >= 70 ? 'text-emerald-600' : 'text-slate-600');
</script>

{#snippet successLine(className?: string)}
	<p class={['font-bold', className, successClass]}>
		{stats.successes}/{stats.total} ({stats.percent}%)
	</p>
{/snippet}

{#if layout === 'inline'}
	<div class="shrink-0 pr-3 text-right">
		{@render successLine('text-sm')}
		<ExecutionModes {stats} class="text-xs text-muted-foreground opacity-80" />
	</div>
{:else if layout === 'card-inline'}
	<div class="text-xs! text-muted-foreground">
		<span class="pr-0.5 capitalize">{m.Success_rate()}</span>
		<span class={['font-semibold', successClass]}>
			{stats.successes}/{stats.total} ({stats.percent}%)
		</span>
		<span class="px-0.5">•</span>
		<span class="pr-0.5">{m.Execution_mode()}</span>
		<ExecutionModes {stats} tag="span" class="font-semibold text-slate-600" />
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
			<ExecutionModes {stats} class="text-sm" />
		</div>
		<T class="text-sm">{label}</T>
	</div>
{/if}
