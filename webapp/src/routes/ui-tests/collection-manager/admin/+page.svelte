<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CollectionManager } from '@/collections-components/manager';
	import List from '@/components/ui-custom/list.svelte';
	import ListItem from '@/components/ui-custom/listItem.svelte';
	import { PUBLIC_POCKETBASE_URL } from '$env/static/public';
	import PocketBase from 'pocketbase';
	import type { TypedPocketBase } from '@/pocketbase/types';
	import CodeDisplay from '$lib/layout/codeDisplay.svelte';

	const pbPromise = init();

	async function init() {
		const pb = new PocketBase(PUBLIC_POCKETBASE_URL) as TypedPocketBase;
		await pb.collection('_superusers').authWithPassword('admin@example.org', 'adminadmin');
		return pb;
	}
</script>

{#await pbPromise then pb}
	<CollectionManager
		collection="users"
		queryOptions={{
			perPage: 6,
			expand: [
				'users_public_keys_via_owner',
				'webauthnCredentials_via_user',
				'orgAuthorizations_via_user'
			]
		}}
		queryAgentOptions={{
			pocketbase: pb
		}}
	>
		{#snippet records({ records })}
			<List>
				{#each records as r}
					<ListItem>
						<CodeDisplay content={JSON.stringify(r, null, 2)} language="json" />
					</ListItem>
				{/each}
			</List>
		{/snippet}
	</CollectionManager>

	<!--  -->

	<CollectionManager
		collection="orgAuthorizations"
		queryOptions={{
			perPage: 6,
			expand: ['role']
		}}
		queryAgentOptions={{
			pocketbase: pb
		}}
	>
		{#snippet records({ records })}
			<List>
				{#each records as r}
					<ListItem>
						<CodeDisplay content={JSON.stringify(r.expand?.role, null, 2)} language="json" />
					</ListItem>
				{/each}
			</List>
		{/snippet}
	</CollectionManager>
{/await}
