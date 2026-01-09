<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import WalletActionTags from '$lib/components/wallet-action-tags.svelte';
	import { getMarketplaceItemData } from '$lib/marketplace/utils.js';

	import { Badge } from '@/components/ui/badge/index.js';
	import { m } from '@/i18n/index.js';

	import type { WalletStepForm } from './wallet-step-form.svelte.js';

	import ItemCard from '../utils/item-card.svelte';
	import SearchInput from '../utils/search-input.svelte';
	import WithEmptyState from '../utils/with-empty-state.svelte';
	import WithLabel from '../utils/with-label.svelte';

	//

	type Props = {
		form: WalletStepForm;
	};

	let { form }: Props = $props();
</script>

{#if form.data.wallet}
	{@const data = getMarketplaceItemData(form.data.wallet)}
	<div class="flex flex-col gap-4 border-b p-4">
		<WithLabel label={m.Wallet()}>
			<ItemCard
				avatar={data.logo}
				title={form.data.wallet.name}
				subtitle={form.data.wallet.organization_name}
				onDiscard={() => form.removeWallet()}
			/>
		</WithLabel>
		{#if form.data.version}
			<WithLabel label={m.Version()}>
				<ItemCard title={form.data.version.tag} onDiscard={() => form.removeVersion()} />
			</WithLabel>
		{/if}
	</div>
{/if}

{#if form.state === 'select-wallet'}
	<WithLabel label={m.Wallet()} class="p-4">
		<SearchInput search={form.walletSearch} />
	</WithLabel>

	<WithEmptyState items={form.foundWallets} emptyText={m.No_results_found()}>
		{#snippet item({ item })}
			<ItemCard
				avatar={getMarketplaceItemData(item).logo}
				title={item.name}
				subtitle={item.organization_name}
				onClick={() => form.selectWallet(item)}
			/>
		{/snippet}
	</WithEmptyState>
{:else if form.state === 'select-version'}
	<WithLabel label={m.Version()} class="p-4 pb-0" />

	<WithEmptyState items={form.foundVersions} emptyText={m.No_wallet_versions_found()}>
		{#snippet item({ item })}
			<ItemCard title={item.tag} onClick={() => form.selectVersion(item)} />
		{/snippet}
	</WithEmptyState>
{:else if form.state === 'select-action'}
	<WithLabel label={m.Wallet_action()} class="p-4">
		<SearchInput search={form.actionSearch} placeholder={m.Search()} />
	</WithLabel>

	<WithEmptyState items={form.foundActions} emptyText={m.No_actions_available()}>
		{#snippet item({ item })}
			<ItemCard title={item.name} onClick={() => form.selectAction(item)}>
				<div class="space-y-2 pt-1">
					{#if !item.published}
						<Badge variant="secondary">
							{m.private()}
						</Badge>
					{/if}
					<WalletActionTags action={item} variant="outline" />
				</div>
			</ItemCard>
		{/snippet}
	</WithEmptyState>
{/if}
