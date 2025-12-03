<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ConformanceCheck } from '$lib/types/checks';

	import DashboardCardManagerTop from '$lib/layout/dashboard-card-manager-top.svelte';
	import DashboardCardManagerUI from '$lib/layout/dashboard-card-manager-ui.svelte';
	import DashboardCard from '$lib/layout/dashboard-card.svelte';

	import type { WalletsResponse } from '@/pocketbase/types';

	import { CollectionManager } from '@/collections-components';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { Separator } from '@/components/ui/separator';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import { setDashboardNavbar } from '../+layout@.svelte';
	import PublicWallets from './public-wallets.svelte';
	import { IDS } from './utils';
	import WalletActionsManager from './wallet-actions-manager.svelte';
	import WalletFormSheet from './wallet-form-sheet.svelte';
	import WalletForm from './wallet-form.svelte';

	//

	let { data } = $props();
	let { organization } = $derived(data);

	setDashboardNavbar({ title: 'Wallets', right: navbarRight });
</script>

<T tag="h2" id={IDS.YOUR_WALLETS}>{m.Your_wallets()}</T>

<CollectionManager
	collection="wallets"
	queryOptions={{
		filter: `owner.id = '${organization.id}'`,
		sort: ['created', 'DESC']
	}}
>
	{#snippet editForm({ record: wallet, closeSheet })}
		<WalletForm walletId={wallet.id} initialData={wallet} onSuccess={() => closeSheet()} />
	{/snippet}

	{#snippet records({ records })}
		<div class="space-y-6">
			{#each records as record (record.id)}
				<DashboardCard
					{record}
					avatar={(w) => (w.logo ? pb.files.getURL(w, w.logo) : w.logo_url)}
					path={[organization.canonified_name, record.canonified_name]}
				>
					{#snippet content()}
						{@const conformanceChecks = record.conformance_checks as
							| ConformanceCheck[]
							| null
							| undefined}
						<div class="flex flex-wrap gap-2">
							{#if conformanceChecks && conformanceChecks.length > 0}
								{#each conformanceChecks as check (check)}
									<Badge
										variant={check.status === 'success'
											? 'secondary'
											: 'destructive'}
									>
										{check.test}
									</Badge>
								{/each}
							{:else}
								<T class="text-sm text-gray-300">
									{m.No_conformance_checks_available()}
								</T>
							{/if}
						</div>

						<Separator />

						{@render walletVersionsManager({
							wallet: record,
							organizationId: organization.id
						})}

						<Separator />

						<WalletActionsManager wallet={record} {organization} />
					{/snippet}
				</DashboardCard>
			{/each}
		</div>
	{/snippet}
</CollectionManager>

<PublicWallets {organization} />

<!--  -->

{#snippet navbarRight()}
	<WalletFormSheet />
{/snippet}

{#snippet walletVersionsManager(props: { wallet: WalletsResponse; organizationId: string })}
	{@const wallet = props.wallet}
	<CollectionManager
		collection="wallet_versions"
		queryOptions={{
			filter: `wallet = '${wallet.id}' && owner.id = '${props.organizationId}'`
		}}
		hide={['empty_state']}
		formFieldsOptions={{
			exclude: ['owner', 'canonified_tag'],
			hide: { wallet: wallet.id },
			placeholders: {
				android_installer: m.Upload_a_new_file(),
				ios_installer: m.Upload_a_new_file(),
				tag: 'e.g. v1.0.0'
			},
			labels: {
				tag: m.Tag(),
				android_installer: m.Android_installer(),
				ios_installer: m.iOS_installer()
			}
		}}
	>
		{#snippet top()}
			<DashboardCardManagerTop
				label={m.Wallet_versions()}
				buttonText={m.Add_new_version()}
				recordCreateOptions={{
					uiOptions: { hideRequiredIndicator: true },
					formTitle: `${m.Wallet()}: ${wallet.name} — ${m.Add_new_version()}`
				}}
			/>
		{/snippet}

		{#snippet records({ records })}
			<DashboardCardManagerUI
				{records}
				nameField="tag"
				hideClone
				path={(r) => [
					organization.canonified_name,
					props.wallet.canonified_name,
					r.canonified_tag
				]}
			>
				{#snippet actions({ record })}
					<div class="flex items-center gap-1">
						{#if record.ios_installer}
							<Badge>iOS</Badge>
						{/if}
						{#if record.android_installer}
							<Badge>Android</Badge>
						{/if}
					</div>
				{/snippet}
			</DashboardCardManagerUI>
		{/snippet}
	</CollectionManager>
{/snippet}
