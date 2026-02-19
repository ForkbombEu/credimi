<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SelfProp } from '$lib/renderable';

	import { Wallet } from '$lib';
	import WalletActionTags from '$lib/components/wallet-action-tags.svelte';
	import { getMarketplaceItemData } from '$lib/marketplace';
	import { ExecutionTarget } from '$lib/pipeline-form/execution-target';

	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';

	import type { WalletActionStepForm } from './wallet-action-step-form.svelte.js';

	import ItemCard from '../_partials/item-card.svelte';
	import SearchInput from '../_partials/search-input.svelte';
	import WithEmptyState from '../_partials/with-empty-state.svelte';
	import WithLabel from '../_partials/with-label.svelte';

	//

	let { self: form }: SelfProp<WalletActionStepForm> = $props();

	const isRunnerGlobal = $derived(ExecutionTarget.hasGlobalRunner());
</script>

{#if form.data.wallet}
	{@const data = getMarketplaceItemData(form.data.wallet)}
	<div class="flex flex-col gap-4 border-b p-4">
		<WithLabel label={m.Wallet()}>
			<ItemCard
				avatar={data.logo}
				title={form.data.wallet.name}
				subtitle={form.data.wallet.organization_name}
				onDiscard={isRunnerGlobal ? undefined : () => form.removeWallet()}
			/>
		</WithLabel>
		{#if form.data.version}
			<WithLabel label={m.Version()}>
				<ItemCard
					title={form.data.version.tag}
					onDiscard={isRunnerGlobal ? undefined : () => form.removeVersion()}
				/>
			</WithLabel>
		{/if}
		{#if form.data.runner}
			<WithLabel label={m.Runner()}>
				{@const title =
					form.data.runner === 'global' ? m.Choose_later() : form.data.runner.name}
				<ItemCard
					{title}
					onDiscard={isRunnerGlobal ? undefined : () => form.removeRunner()}
				/>
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
{:else if form.state === 'select-runner'}
	<WithLabel label={m.Runner()} class="p-4">
		<SearchInput search={form.runnerSearch} />
	</WithLabel>

	{#if ExecutionTarget.hasUndefinedRunner()}
		<div class="px-4">
			<ItemCard title={m.Choose_later()} onClick={() => form.selectRunner('global')} />
		</div>
	{/if}
	<WithEmptyState items={form.foundRunners} emptyText={m.No_runners_found()}>
		{#snippet item({ item })}
			<ItemCard title={item.name} onClick={() => form.selectRunner(item)}>
				{#snippet right()}
					{#if !item.published}
						<Badge variant="secondary">
							{m.private()}
						</Badge>
					{/if}
				{/snippet}
			</ItemCard>
		{/snippet}
	</WithEmptyState>
{:else if form.state === 'select-action'}
	<WithLabel label={m.Wallet_action()} class="p-4">
		<SearchInput search={form.actionSearch} placeholder={m.Search()} />
	</WithLabel>

	<WithEmptyState items={form.foundActions} emptyText={m.No_actions_available()}>
		{#snippet item({ item })}
			<ItemCard title={item.name} onClick={() => form.selectAction(item)}>
				{#snippet beforeContent()}
					{@const category = Wallet.Action.getCategoryLabel(item)}
					{#if category}
						<T class="text-xs text-muted-foreground">{category}</T>
					{/if}
				{/snippet}

				{#snippet afterContent()}
					<WalletActionTags action={item} variant="secondary" containerClass="pt-2">
						{#if !item.published}
							<Badge variant="outline">
								{m.private()}
							</Badge>
						{/if}
					</WalletActionTags>
				{/snippet}
			</ItemCard>
		{/snippet}
	</WithEmptyState>
{/if}
