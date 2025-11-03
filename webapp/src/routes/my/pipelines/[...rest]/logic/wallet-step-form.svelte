<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { getMarketplaceItemData } from '$lib/marketplace/utils.js';
	import { XIcon } from 'lucide-svelte';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n/index.js';

	import type { WalletStepForm } from './wallet.svelte.js';

	import ItemCard from './item-card.svelte';
	import SearchInput from './utils/search-input.svelte';
	import WithEmptyState from './utils/with-empty-state.svelte';

	//

	type Props = {
		form: WalletStepForm;
	};

	let { form }: Props = $props();
</script>

<div class="space-y-4">
	{#if form.data.wallet}
		{@const data = getMarketplaceItemData(form.data.wallet)}
		<div class="flex flex-col gap-2 border-y py-4">
			<div class="space-y-1">
				<T class="text-muted-foreground text-sm">{m.Wallet()}</T>
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
			</div>

			{#if form.data.version}
				<div class="space-y-1">
					<T class="text-muted-foreground text-sm">{m.Version()}</T>
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
				</div>
			{/if}
		</div>
	{/if}

	{#if form.state === 'select-wallet'}
		<div class="space-y-4">
			<SearchInput search={form.walletSearch} />

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
		<WithEmptyState items={form.foundVersions} emptyText={m.No_wallet_versions_found()}>
			{#snippet item({ item })}
				<ItemCard title={item.tag} onClick={() => form.selectVersion(item)} />
			{/snippet}
		</WithEmptyState>
	{:else if form.state === 'select-action'}
		<div class="space-y-4">
			<SearchInput search={form.actionSearch} placeholder="Search for an action" />

			<div class="flex flex-col gap-2">
				{#each form.foundActions as action (action.id)}
					<ItemCard title={action.name} onClick={() => form.selectAction(action)} />
				{/each}
			</div>
		</div>
	{/if}
</div>
