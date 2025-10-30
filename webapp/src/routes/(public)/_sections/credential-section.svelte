<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import CredentialCard from '$lib/layout/credentialCard.svelte';
	import PageGrid from '$lib/layout/pageGrid.svelte';

	import { CollectionManager } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	//

	const MAX_ITEMS = 3;
</script>

,
<div class="space-y-6">
	<div class="flex items-center justify-between">
		<T tag="h3">{m.Find_solutions()}</T>
		<Button variant="default" href="/marketplace">{m.Explore_Marketplace()}</Button>
	</div>

	<CollectionManager
		collection="credentials"
		queryOptions={{
			perPage: MAX_ITEMS,
			filter: `published = true`
		}}
		hide={['pagination']}
	>
		{#snippet records({ records })}
			<PageGrid>
				{#each records as credential, i (credential.id)}
					{@const isLast = i == MAX_ITEMS - 1}
					<CredentialCard {credential} class={isLast ? 'hidden lg:flex' : ''} />
				{/each}
			</PageGrid>
		{/snippet}
	</CollectionManager>
</div>
