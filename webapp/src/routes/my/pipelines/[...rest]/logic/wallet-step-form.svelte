<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { getMarketplaceItemData } from '$lib/marketplace/utils.js';
	import { XIcon } from 'lucide-svelte';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n/index.js';

	import type { WalletStepForm } from './wallet.svelte.js';

	import ItemCard from './utils/item-card.svelte';
	import SearchInput from './utils/search-input.svelte';
	import WithEmptyState from './utils/with-empty-state.svelte';
	import WithLabel from './utils/with-label.svelte';

	//

	type Props = {
		form: WalletStepForm;
	};

	let { form }: Props = $props();
</script>

<div class="space-y-4">
	{#if form.data.wallet}
		{@const data = getMarketplaceItemData(form.data.wallet)}
		<div class="flex flex-col gap-2 border-b pb-4">
			<WithLabel label={m.Wallet()}>
				<ItemCard
					avatar={data.logo}
					title={form.data.wallet.name}
					subtitle={form.data.wallet.organization_name}
				>
					{#snippet right()}
						<IconButton
							icon={XIcon}
							variant="ghost"
							size="xs"
							class="hover:bg-gray-200"
							onclick={() => form.removeWallet()}
						/>
					{/snippet}
				</ItemCard>
			</WithLabel>

			{#if form.data.version}
				<WithLabel label={m.Version()}>
					<ItemCard title={form.data.version.tag}>
						{#snippet right()}
							<IconButton
								icon={XIcon}
								variant="ghost"
								size="xs"
								class="hover:bg-gray-200"
								onclick={() => form.removeVersion()}
							/>
						{/snippet}
					</ItemCard>
				</WithLabel>
			{/if}
		</div>
	{/if}

	{#if form.state === 'select-wallet'}
		<div class="space-y-4">
			<WithLabel label={m.Wallet()}>
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
		</div>
	{:else if form.state === 'select-version'}
		<WithLabel label={m.Version()}>
			<WithEmptyState items={form.foundVersions} emptyText={m.No_wallet_versions_found()}>
				{#snippet item({ item })}
					<ItemCard title={item.tag} onClick={() => form.selectVersion(item)} />
				{/snippet}
			</WithEmptyState>
		</WithLabel>
	{:else if form.state === 'select-action'}
		<div class="space-y-4">
			<WithLabel label={m.Wallet_action()}>
				<SearchInput search={form.actionSearch} placeholder={m.Search()} />
			</WithLabel>

			<WithEmptyState items={form.foundActions} emptyText={m.No_actions_available()}>
				{#snippet item({ item })}
					<ItemCard title={item.name} onClick={() => form.selectAction(item)} />
				{/snippet}
			</WithEmptyState>
		</div>
	{/if}
</div>
