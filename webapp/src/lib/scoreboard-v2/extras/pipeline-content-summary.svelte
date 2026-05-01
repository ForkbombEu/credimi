<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { entities, type EntityData } from '$lib/global';

	import A from '@/components/ui-custom/a.svelte';

	import type { ScoreboardRow } from '../types';

	import Avatar from '../columns/partials/avatar.svelte';
	import { getRelatedEntityHref, type RelatedEntity } from '../columns/partials/types';

	//

	type Props = {
		results: ScoreboardRow;
	};

	let { results }: Props = $props();

	const wallets = $derived(results.expand?.wallets ?? []);
	const issuers = $derived(results.expand?.issuers ?? []);
	const verifiers = $derived(results.expand?.verifiers ?? []);
	const conformanceChecks = $derived(results.conformance_checks ?? []);

	const hasRelatedEntities = $derived(
		wallets.length > 0 || issuers.length > 0 || verifiers.length > 0
	);
</script>

{#if hasRelatedEntities}
	<div class="flex flex-wrap items-center gap-4">
		{#each wallets as wallet (wallet.id)}
			{@render entity(wallet, entities.wallets)}
		{/each}
		{#each issuers as issuer (issuer.id)}
			{@render entity(issuer, entities.credential_issuers, results.credentials.length)}
		{/each}
		{#each verifiers as verifier (verifier.id)}
			{@render entity(verifier, entities.verifiers, results.use_case_verifications.length)}
		{/each}
	</div>
{/if}

{#snippet entity(entity: RelatedEntity, displayData: EntityData, count?: number)}
	<div class="flex items-center gap-2">
		<Avatar record={entity} />
		<div class="flex flex-col text-xs">
			<A href={getRelatedEntityHref(entity)} class="max-w-[15ch] truncate">
				{entity.name}
			</A>
			<p class={[displayData.classes.text]}>
				<displayData.icon class="inline-block size-3 -translate-y-px" />
				{displayData.labels.singular}
				{#if count}
					<span>• {count}</span>
				{/if}
			</p>
		</div>
	</div>
{/snippet}
