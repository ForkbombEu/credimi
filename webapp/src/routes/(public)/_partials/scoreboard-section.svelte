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
			records: res.items
		};
	}
</script>

<script lang="ts">
	import { entities, EntityTag, type EntityData } from '$lib/global';
	import CardLink from '$lib/layout/card-link.svelte';
	import Avatar from '$lib/scoreboard-v2/columns/partials/avatar.svelte';
	import {
		getRelatedEntityHref,
		type RelatedEntity
	} from '$lib/scoreboard-v2/columns/partials/types';

	import A from '@/components/ui-custom/a.svelte';

	type Props = Awaited<ReturnType<typeof loadScoreboardSummary>>;

	let { records }: Props = $props();
</script>

<div class="space-y-2">
	{#each records as record (record.id)}
		{@const wallets = record.expand?.wallets ?? []}
		{@const issuers = record.expand?.issuers ?? []}
		{@const verifiers = record.expand?.verifiers ?? []}
		{@const hasRelatedEntities =
			wallets.length > 0 || issuers.length > 0 || verifiers.length > 0}

		<CardLink
			href={getRelatedEntityHref(record.expand!.pipeline!)}
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

			{#if hasRelatedEntities}
				<div class="flex items-center gap-4">
					{#each wallets as wallet (wallet.id)}
						{@render entity(wallet, entities.wallets)}
					{/each}
					{#each issuers as issuer (issuer.id)}
						{@render entity(issuer, entities.credential_issuers)}
					{/each}
					{#each verifiers as verifier (verifier.id)}
						{@render entity(verifier, entities.verifiers)}
					{/each}
				</div>
			{/if}
		</CardLink>
	{/each}
</div>

{#snippet entity(entity: RelatedEntity, displayData: EntityData)}
	<div class="flex items-center gap-2">
		<Avatar record={entity} />
		<div class="flex flex-col text-xs">
			<A href={getRelatedEntityHref(entity)} class="max-w-[15ch] truncate">
				{entity.name}
			</A>
			<p class="text-muted-foreground">{displayData.labels.singular}</p>
		</div>
	</div>
{/snippet}
