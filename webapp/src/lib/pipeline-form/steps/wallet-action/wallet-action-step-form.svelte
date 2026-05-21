<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SelfProp } from '$lib/renderable';

	import { ExternalLinkIcon } from '@lucide/svelte';
	import { Wallet } from '$lib';
	import AndroidLogo from '$lib/components/android-logo.svelte';
	import AppleLogo from '$lib/components/apple-logo.svelte';
	import WalletActionTags from '$lib/components/wallet-action-tags.svelte';
	import { getHubItemData } from '$lib/hub';
	import { ExecutionTarget } from '$lib/pipeline-form/execution-target';
	import { bindRunnerCatalogSearch } from '$lib/pipeline/runner/runner-select-catalog.svelte.js';
	import RunnerSelectList from '$lib/pipeline/runner/runner-select-list.svelte';

	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';

	import ItemCard from '../_partials/item-card.svelte';
	import SearchInput from '../_partials/search-input.svelte';
	import WithEmptyState from '../_partials/with-empty-state.svelte';
	import WithLabel from '../_partials/with-label.svelte';
	import {
		getRunnerLabel,
		getVersionLabel,
		type WalletActionStepForm
	} from './wallet-action-step-form.svelte.js';

	//

	let { self: form }: SelfProp<WalletActionStepForm> = $props();

	const isRunnerGlobal = $derived(ExecutionTarget.hasGlobalRunner());

	const runnerCatalog = bindRunnerCatalogSearch({
		search: form.runnerSearch
	});
</script>

{#snippet chooseRunnerLater()}
	{#if ExecutionTarget.hasUndefinedRunner()}
		<div class="px-4">
			<ItemCard title={m.Choose_later()} onClick={() => form.selectRunner('global')} />
		</div>
	{/if}
{/snippet}

{#if form.data.wallet}
	{@const data = getHubItemData(form.data.wallet)}
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
					title={getVersionLabel(form.data.version)}
					onDiscard={isRunnerGlobal ? undefined : () => form.removeVersion()}
				/>
			</WithLabel>
		{/if}
		{#if form.data.runner}
			<WithLabel label={m.Runner()}>
				<ItemCard
					title={getRunnerLabel(form.data.runner)}
					onDiscard={isRunnerGlobal ? undefined : () => form.removeRunner()}
				/>
			</WithLabel>
		{/if}
		{#if form.data.action}
			<WithLabel label={m.Wallet_action()}>
				<ItemCard title={form.data.action.name} onDiscard={() => form.removeAction()} />
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
				avatar={getHubItemData(item).logo}
				title={item.name}
				subtitle={item.organization_name}
				onClick={() => form.selectWallet(item)}
			/>
		{/snippet}
	</WithEmptyState>
{:else if form.state === 'select-version'}
	<WithLabel label={m.Version()} class="p-4" />

	<div class="px-4">
		<ItemCard
			title={m.Install_from_external_source()}
			onClick={() => form.selectExternalVersion()}
		>
			{#snippet titleRight()}
				<span class="ml-0.5 inline-flex translate-0.5 gap-1 text-gray-300">
					<ExternalLinkIcon size={16} class="stroke-2" />
				</span>
			{/snippet}
		</ItemCard>
	</div>

	<WithEmptyState items={form.foundVersions} emptyText={m.No_wallet_versions_found()}>
		{#snippet item({ item })}
			<ItemCard title={item.tag} onClick={() => form.selectVersion(item)}>
				{#snippet titleRight()}
					<span class="ml-0.5 inline-flex translate-0.5 gap-1 text-gray-300">
						{#if item.android_installer}
							<AndroidLogo size={16} />
						{/if}
						{#if item.ios_installer}
							<AppleLogo size={16} />
						{/if}
					</span>
				{/snippet}
			</ItemCard>
		{/snippet}
	</WithEmptyState>
{:else if form.state === 'select-runner'}
	<WithLabel label={m.Runner()} class="p-4">
		<SearchInput search={form.runnerSearch} />
	</WithLabel>

	<RunnerSelectList
		presentation="minimal"
		foundRunners={runnerCatalog.foundRunners}
		catalogLoading={runnerCatalog.catalogLoading}
		scrollable
		prepend={chooseRunnerLater}
		onSelect={(item) => form.selectRunner(item)}
	/>
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
