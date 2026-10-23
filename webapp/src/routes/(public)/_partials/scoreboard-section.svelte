<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ScoreboardRow } from '$lib/scoreboard/types';

	import { ClockIcon, SmartphoneIcon } from '@lucide/svelte';
	import { entities } from '$lib/global';
	import { fromEnrichedRecord } from '$lib/pipeline/execution-artifacts';
	import ExecutionArtifactsPreview from '$lib/pipeline/results/execution-artifacts-preview.svelte';
	import * as conformanceChecks from '$lib/scoreboard/columns/conformance-checks.svelte';
	import * as customIntegrations from '$lib/scoreboard/columns/custom-integrations.svelte';
	import * as issuers from '$lib/scoreboard/columns/issuers.svelte';
	import * as verifiers from '$lib/scoreboard/columns/verifiers.svelte';
	import * as wallets from '$lib/scoreboard/columns/wallets.svelte';
	import * as EntityDisplay from '$lib/scoreboard/entity-display';
	import ExecutionModes from '$lib/scoreboard/extras/execution-modes.svelte';
	import { formatExecutionTimestamp } from '$lib/scoreboard/extras/format-date';
	import { fromScoreboardRow } from '$lib/scoreboard/extras/from-scoreboard-row';
	import { hasVisiblePipeline } from '$lib/scoreboard/records';
	import ScorePill from '$lib/scoreboard/score-pill.svelte';
	import { getPath } from '$lib/utils';
	import { fromStore } from 'svelte/store';

	import { m } from '@/i18n';
	import { currentUser } from '@/pocketbase';

	type Props = {
		records: ScoreboardRow[];
	};

	let { records }: Props = $props();

	const user = fromStore(currentUser);
	const visibleRecords = $derived(records.filter(hasVisiblePipeline));

	/** The same groups as the expanded scoreboard row, without the empty ones. */
	function groups(record: ScoreboardRow) {
		return [
			{ label: entities.wallets.labels.plural, items: wallets.column.fn(record) },
			{ label: m.Issuance(), items: issuers.column.fn(record) },
			{ label: m.Presentations(), items: verifiers.column.fn(record) },
			{
				label: entities.conformance_checks.labels.plural,
				items: conformanceChecks.column.fn(record)
			},
			{
				label: entities.custom_checks.labels.plural,
				items: customIntegrations.column.fn(record)
			}
		].filter((group) => group.items.length > 0);
	}

	function artifacts(record: ScoreboardRow) {
		const execution = record.expanded_data?.latest_execution;
		return execution
			? fromEnrichedRecord(execution as Parameters<typeof fromEnrichedRecord>[0])
			: undefined;
	}
</script>

<div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
	{#each visibleRecords as record (record.id)}
		{@const pipeline = record.expanded_data!.pipeline!}
		{@const evidence = artifacts(record)}
		{@const stats = fromScoreboardRow(record)}
		{@const devices = record.expanded_data?.mobile_devices?.length ?? 0}
		{@const lastRun = record.expanded_data?.latest_execution?.created}
		<article
			class="flex min-w-0 flex-col gap-5 rounded-[10px] border border-border bg-card p-5"
		>
			<header class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
				<div class="min-w-0">
					{#if pipeline.published}
						<a
							href="/hub/pipelines/{getPath(pipeline)}"
							class="line-clamp-2 font-semibold text-primary hover:underline"
						>
							{pipeline.name}
						</a>
					{:else}
						<p class="line-clamp-2 font-semibold">{pipeline.name}</p>
					{/if}
					<div class="mt-2 flex flex-wrap items-center gap-2">
						<ScorePill percent={record.success_rate} total={record.total_runs ?? 0} />
						<span class="text-xs text-muted-foreground">
							{m.scoreboard_runs_passed({
								successes: record.total_successes ?? 0,
								total: record.total_runs ?? 0
							})}
						</span>
					</div>
				</div>
				{#if evidence}
					<div class="shrink-0">
						<ExecutionArtifactsPreview
							artifacts={evidence}
							presentation="preview"
							previewClass="size-12!"
							hideLogs
						/>
					</div>
				{/if}
			</header>

			<dl class="grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2">
				{#each groups(record) as group (group.label)}
					<div class="flex min-w-0 flex-col gap-2">
						<dt
							class="text-[11px] font-semibold tracking-[0.14em] text-muted-foreground uppercase"
						>
							{group.label}
						</dt>
						<dd><EntityDisplay.List items={group.items} layout="compact" /></dd>
					</div>
				{/each}
			</dl>

			<footer
				class="mt-auto flex flex-wrap items-center gap-x-5 gap-y-2 border-t border-border pt-3 text-xs text-muted-foreground"
			>
				{#if stats}
					<ExecutionModes {stats} tag="span" class="tabular-nums" />
				{/if}
				{#if record.minimum_running_time}
					<span class="inline-flex items-center gap-1.5">
						<ClockIcon class="size-3.5" aria-hidden="true" />
						{record.minimum_running_time}
						{m.Min_running_time()}
					</span>
				{/if}
				{#if devices > 0}
					<span class="inline-flex items-center gap-1.5">
						<SmartphoneIcon class="size-3.5" aria-hidden="true" />
						{devices}
						{m.Devices()}
					</span>
				{/if}
				{#if lastRun}
					<span class="ml-auto">
						{m.scoreboard_last_run()}
						<time class="font-mono" datetime={lastRun}>
							{formatExecutionTimestamp(lastRun, user.current?.Timezone)}
						</time>
					</span>
				{/if}
			</footer>
		</article>
	{/each}
</div>
