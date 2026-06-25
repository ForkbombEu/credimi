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
	import { type HubItem, getHubItemData } from '$lib/hub';
	import { getHubItemTypeFilter } from '$lib/hub/utils.js';
	import { ExecutionTarget } from '$lib/pipeline-form/execution-target';
	import { bindRunnerCatalogSearch } from '$lib/pipeline/runner/runner-select-catalog.svelte.js';
	import RunnerSelectList from '$lib/pipeline/runner/runner-select-list.svelte';

	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';

	import ItemCard from '../_partials/item-card.svelte';
	import SearchInput from '../_partials/search-input.svelte';
	import StepCollectionPicker from '../_partials/step-collection-picker.svelte';
	import WithLabel from '../_partials/with-label.svelte';
	import WalletActionForm from './wallet-action-form.svelte';
	import {
		getRunnerLabel,
		getVersionLabel,
		type WalletActionStepForm
	} from './wallet-action-step-form.svelte.js';

	//

	let { self: form }: SelfProp<WalletActionStepForm> = $props();

	const isTargetLocked = $derived(form.isTargetLocked);
	const showChooseRunnerLater = $derived(
		ExecutionTarget.hasUndefinedRunner() && !form.hasDistinctOtherMobileWallets
	);

	const runnerCatalog = bindRunnerCatalogSearch({
		search: form.runnerSearch
	});
</script>

{#snippet chooseRunnerLater()}
	{#if showChooseRunnerLater}
		<div class="px-4">
			<ItemCard title={m.Choose_later()} onClick={() => form.selectRunner('global')} />
		</div>
	{/if}
{/snippet}

{#if form.data.wallet}
	{@const data = getHubItemData(form.data.wallet)}
	{@const walletAction = form.data.action}
	<div class="flex flex-col gap-4 border-b p-4">
		<WithLabel label={m.Wallet()}>
			<ItemCard
				avatar={data.logo}
				title={form.data.wallet.name}
				subtitle={form.data.wallet.organization_name}
				onDiscard={isTargetLocked ? undefined : () => form.removeWallet()}
			/>
		</WithLabel>
		{#if form.data.version}
			<WithLabel label={m.Version()}>
				<ItemCard
					title={getVersionLabel(form.data.version)}
					onDiscard={isTargetLocked ? undefined : () => form.removeVersion()}
				/>
			</WithLabel>
		{/if}
		{#if form.data.runner}
			<WithLabel label={m.Runner()}>
				<ItemCard
					title={getRunnerLabel(form.data.runner)}
					onDiscard={isTargetLocked ? undefined : () => form.removeRunner()}
				/>
			</WithLabel>
		{/if}
		{#if walletAction}
			<WithLabel label={m.Wallet_action()}>
				<ItemCard title={walletAction.name} onDiscard={() => form.removeAction()} />
				{#snippet labelRight()}
					<WalletActionForm {walletAction} />
				{/snippet}
			</WithLabel>
		{/if}
	</div>
{/if}

{#if form.state === 'select-wallet'}
	<StepCollectionPicker
		collection="hub_items"
		label={m.Wallet()}
		queryOptions={{
			filter: getHubItemTypeFilter('wallets'),
			searchFields: ['name']
		}}
		onSelect={(record) => form.selectWallet(record as HubItem)}
	>
		{#snippet item({ record, onSelect })}
			{@const item = record as HubItem}
			{@const data = getHubItemData(item)}
			<ItemCard
				avatar={data.logo}
				title={item.name}
				subtitle={item.organization_name}
				onClick={() => onSelect(record)}
			/>
		{/snippet}
	</StepCollectionPicker>
{:else if form.state === 'select-version'}
	<StepCollectionPicker
		collection="wallet_versions"
		label={m.Version()}
		queryOptions={{
			filter: `wallet = '${form.data.wallet!.id}'`,
			sort: ['tag', 'DESC']
		}}
		emptyText={m.No_wallet_versions_found()}
		onSelect={(record) => form.selectVersion(record)}
	>
		{#snippet prepend()}
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
		{/snippet}
		{#snippet item({ record, onSelect })}
			<ItemCard title={record.tag} onClick={() => onSelect(record)}>
				{#snippet titleRight()}
					<span class="ml-0.5 inline-flex translate-0.5 gap-1 text-gray-300">
						{#if record.android_installer}
							<AndroidLogo size={16} />
						{/if}
						{#if record.ios_installer}
							<AppleLogo size={16} />
						{/if}
					</span>
				{/snippet}
			</ItemCard>
		{/snippet}
	</StepCollectionPicker>
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
	<StepCollectionPicker
		collection="wallet_actions"
		label={m.Wallet_action()}
		queryOptions={{
			filter: `wallet = '${form.data.wallet!.id}'`,
			searchFields: ['name', 'canonified_name'],
			sort: ['category', 'ASC']
		}}
		emptyText={m.No_actions_available()}
		onSelect={(record) => form.selectAction(record)}
	>
		{#snippet item({ record, onSelect })}
			<ItemCard title={record.name} onClick={() => onSelect(record)}>
				{#snippet beforeContent()}
					{@const category = Wallet.Action.getCategoryLabel(record)}
					{#if category}
						<T class="text-xs text-muted-foreground">{category}</T>
					{/if}
				{/snippet}
				{#snippet afterContent()}
					<WalletActionTags action={record} variant="secondary" containerClass="pt-2">
						{#if !record.published}
							<Badge variant="outline">
								{m.private()}
							</Badge>
						{/if}
					</WalletActionTags>
				{/snippet}
			</ItemCard>
		{/snippet}
	</StepCollectionPicker>
{/if}
