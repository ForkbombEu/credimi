<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ScoreboardRow } from '$lib/scoreboard/types';

	import { entities, EntityTag } from '$lib/global';
	import { getPath } from '$lib/utils';
	import CardLink from '$lib/layout/card-link.svelte';
	import * as EntityDisplay from '$lib/scoreboard/entity-display';
	import PipelineContentSummary from '$lib/scoreboard/extras/pipeline-content-summary.svelte';

	type Props = {
		records: ScoreboardRow[];
	};

	let { records }: Props = $props();

	const cardClass = 'flex flex-col flex-wrap justify-between gap-4 p-3! md:flex-row';
</script>

{#snippet scoreboardContent(record: ScoreboardRow)}
	{@const pipeline = record.expand?.pipeline}
	<div class="flex flex-col items-start gap-2 md:flex-row md:items-center">
		<EntityTag data={entities.pipelines} />
		<div class="text-sm">
			<p>
				{#if pipeline}
					{pipeline.name}
				{:else}
					<EntityDisplay.Na />
				{/if}
			</p>
			<p class="font-bold">
				• {record.total_successes} / {record.total_runs} ({record.success_rate}%)
			</p>
		</div>
	</div>

	<PipelineContentSummary results={record} />
{/snippet}

<div class="space-y-2">
	{#each records as record (record.id)}
		{@const pipeline = record.expand?.pipeline}
		{#if pipeline?.published}
			<CardLink href={`/hub/pipelines/${getPath(pipeline)}`} class={cardClass}>
				{@render scoreboardContent(record)}
			</CardLink>
		{:else}
			<div
				class={[
					'block rounded-lg border border-primary bg-card text-card-foreground shadow-sm',
					cardClass
				]}
			>
				{@render scoreboardContent(record)}
			</div>
		{/if}
	{/each}
</div>
