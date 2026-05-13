<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import { PocketbaseQueryAgent } from '@/pocketbase/query';

	const queryAgent = new PocketbaseQueryAgent({
		collection: 'pipeline_scoreboard_cache',
		expand: ['pipeline', 'wallets', 'issuers', 'verifiers', 'latest_successful_execution']
	});

	export async function loadScoreboardSummary() {
		const res = await queryAgent.getList(1, 5, {
			sort: '@random'
		});
		return {
			records: res.items as ScoreboardRow[]
		};
	}
</script>

<script lang="ts">
	import type { ScoreboardRow } from '$lib/scoreboard/types';

	import { Scoreboard } from '$lib';
	import { entities, EntityTag } from '$lib/global';
	import CardLink from '$lib/layout/card-link.svelte';
	import PipelineContentSummary from '$lib/scoreboard/extras/pipeline-content-summary.svelte';

	type Props = Awaited<ReturnType<typeof loadScoreboardSummary>>;

	let { records }: Props = $props();
</script>

<div class="space-y-2">
	{#each records as record (record.id)}
		<CardLink
			href={Scoreboard.EntityDisplay.getPocketbaseEntityHref(record.expand!.pipeline!)}
			class="flex flex-col flex-wrap justify-between gap-4 p-3! md:flex-row"
		>
			<div class="flex flex-col items-start gap-2 md:flex-row md:items-center">
				<EntityTag data={entities.pipelines} />
				<div class="text-sm">
					<p>{record.expand?.pipeline?.name}</p>
					<p class="font-bold">
						• {record.total_successes} / {record.total_runs} ({record.success_rate}%)
					</p>
				</div>
			</div>

			<PipelineContentSummary results={record} />
		</CardLink>
	{/each}
</div>
