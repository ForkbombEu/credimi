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

	import { IDS } from '../_partials/sidebar-data.svelte.js';
	import WalletActionsManager from './wallet-actions-manager.svelte';

	//

	type Props = {
		organization: OrganizationsResponse;
	};

	let { organization }: Props = $props();
</script>

<CollectionManager
	collection="wallets"
	queryOptions={{ filter: `owner.id != '${organization.id}'` }}
>
	{#snippet top({ Search })}
		<div class="flex items-center justify-between gap-12">
			<div class="scroll-mt-6" id={IDS.PUBLIC}>
				<T tag="h4" class="pb-1!">{m.Public_wallets()}</T>
				<T class="text-muted-foreground">{m.public_wallets_description()}</T>
			</div>

			<Search containerClass="grow max-w-sm" />
		</div>
	{/snippet}

	{#snippet records({ records })}
		<div class="space-y-6">
			{#each records as record (record.id)}
				<DashboardCard
					{record}
					avatar={(w) => (w.logo ? pb.files.getURL(w, w.logo) : w.logo_url)}
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
