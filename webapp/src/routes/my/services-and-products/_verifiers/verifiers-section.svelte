<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CollectionManager } from '@/collections-components';
	import { m } from '@/i18n';

	import VerifierCard from './verifier-card.svelte';
	import { settings } from './verifier-form-settings.svelte';

	//

	type Props = {
		organizationId: string;
		id?: string;
	};

	let { organizationId, id }: Props = $props();
</script>

<CollectionManager
	collection="verifiers"
	queryOptions={{
		filter: `owner.id = '${organizationId}'`,
		expand: ['use_cases_verifications_via_verifier'],
		sort: ['created', 'DESC']
	}}
	formFieldsOptions={settings}
>
	{#snippet top({ Header })}
		<Header title="Verifiers" {id}>
			{#snippet buttonContent()}
				{m.Create_verifier()}
			{/snippet}
		</Header>
	{/snippet}

	{#snippet records({ records })}
		<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
			{#each records as verifier, index}
				{@const useCasesVerifications =
					verifier.expand?.use_cases_verifications_via_verifier ?? []}
				<VerifierCard
					bind:verifier={records[index]}
					{useCasesVerifications}
					{organizationId}
				/>
			{/each}
		</div>
	{/snippet}
</CollectionManager>
