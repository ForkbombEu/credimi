<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import DashboardCard from '$lib/layout/dashboard-card.svelte';

	import type { OrganizationsResponse } from '@/pocketbase/types';

	import { CollectionManager } from '@/collections-components';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import WalletActionsManager from './wallet-actions-manager.svelte';

	//

	type Props = {
		organization: OrganizationsResponse;
	};

	let { organization }: Props = $props();
</script>

<div class="mt-8">
	<T tag="h2">{m.Public_wallets()}</T>
	<T class="text-muted-foreground">{m.public_wallets_description()}</T>
</div>

<CollectionManager
	collection="wallets"
	queryOptions={{ filter: `owner.id != '${organization.id}'` }}
>
	{#snippet records({ records })}
		<div class="space-y-6">
			{#each records as record (record.id)}
				<DashboardCard
					{record}
					avatar={(w) => (w.logo ? pb.files.getURL(w, w.logo) : w.logo_url)}
					path={[organization.canonified_name, record.canonified_name]}
					hideActions
				>
					{#snippet content()}
						<WalletActionsManager wallet={record} {organization} />
					{/snippet}
				</DashboardCard>
			{/each}
		</div>
	{/snippet}</CollectionManager
>
