<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ExternalLinkIcon } from '@lucide/svelte';
	import AndroidLogo from '$lib/components/android-logo.svelte';
	import AppleLogo from '$lib/components/apple-logo.svelte';
	import { getMarketplaceItemData } from '$lib/marketplace';

	import type { WalletVersionsResponse } from '@/pocketbase/types';

	import Dialog from '@/components/ui-custom/dialog.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase/index.js';

	import type { StepsBuilder } from '../steps-builder.svelte.js';

	import ItemCard from '../../steps/_partials/item-card.svelte';
	import WithEmptyState from '../../steps/_partials/with-empty-state.svelte';
	import WithLabel from '../../steps/_partials/with-label.svelte';
	import {
		EXTERNAL_VERSION,
		type SelectedVersion,
		type WalletActionStepData
	} from '../../steps/wallet-action/wallet-action-step-form.svelte.js';
	import { getBulkWalletVersionContext } from './bulk-wallet-version-context.js';

	type Props = {
		builder: StepsBuilder;
	};

	let { builder }: Props = $props();

	const bulkContext = $derived(getBulkWalletVersionContext(builder.steps));

	/** First mobile step’s version (all mobile steps share the same when bulkContext is set). */
	const currentVersionProbe = $derived.by(() => {
		const ctx = bulkContext;
		if (!ctx) return { isExternal: false, recordId: null as string | null };
		const tuple = builder.steps[ctx.mobileIndices[0]!];
		if (!tuple) return { isExternal: false, recordId: null };
		const data = tuple[1] as unknown as WalletActionStepData;
		const v = data.version;
		if (v === EXTERNAL_VERSION) return { isExternal: true, recordId: null };
		if (v && typeof v === 'object' && 'id' in v) {
			return { isExternal: false, recordId: v.id };
		}
		return { isExternal: false, recordId: null };
	});

	let walletVersionDialogOpen = $state(false);
	let foundVersions = $state<WalletVersionsResponse[]>([]);
	let versionsLoading = $state(false);
	let versionsLoadError = $state<string | null>(null);

	$effect(() => {
		const ctx = bulkContext;
		if (!walletVersionDialogOpen || !ctx) {
			if (!walletVersionDialogOpen) {
				foundVersions = [];
				versionsLoadError = null;
				versionsLoading = false;
			}
			return;
		}

		versionsLoading = true;
		versionsLoadError = null;

		let cancelled = false;
		pb.collection('wallet_versions')
			.getFullList({
				filter: `wallet = "${ctx.wallet.id}"`,
				requestKey: null
			})
			.then((list) => {
				if (!cancelled) {
					foundVersions = list;
					versionsLoading = false;
				}
			})
			.catch((e: unknown) => {
				if (!cancelled) {
					versionsLoadError = e instanceof Error ? e.message : String(e);
					versionsLoading = false;
				}
			});

		return () => {
			cancelled = true;
		};
	});

	function applyVersionAndClose(
		version: SelectedVersion,
		closeDialog: () => void | Promise<void>
	) {
		builder.applyBulkWalletVersion(version);
		void closeDialog();
	}

	function isCurrentWalletVersionRow(item: WalletVersionsResponse) {
		return (
			!currentVersionProbe.isExternal &&
			currentVersionProbe.recordId !== null &&
			currentVersionProbe.recordId === item.id
		);
	}
</script>

{#if bulkContext}
	<button
		class="shrink-0 text-xs text-primary hover:cursor-pointer hover:underline"
		onclick={() => (walletVersionDialogOpen = true)}
	>
		{m.Change_wallet_version()}
	</button>
{/if}

<Dialog
	bind:open={walletVersionDialogOpen}
	hideTrigger
	title={m.Change_wallet_version_modal_title()}
	description={m.Change_wallet_version_modal_description()}
	contentClass="max-h-[min(80vh,560px)] overflow-y-auto"
>
	{#snippet content({ closeDialog })}
		{#if bulkContext}
			{@const walletData = getMarketplaceItemData(bulkContext.wallet)}
			<div class="flex flex-col gap-4 py-2">
				<div class="flex flex-col gap-2 border-b pb-4">
					<WithLabel label={m.Wallet()}>
						<ItemCard
							avatar={walletData.logo}
							title={bulkContext.wallet.name}
							subtitle={bulkContext.wallet.organization_name}
						/>
					</WithLabel>
				</div>

				{#if versionsLoading}
					<p class="text-sm text-muted-foreground">{m.Loading()}</p>
				{:else if versionsLoadError}
					<p class="text-sm text-destructive">{versionsLoadError}</p>
				{:else}
					<WithLabel label={m.Version()} />

					<ItemCard
						title={m.Install_from_external_source()}
						onClick={() => applyVersionAndClose(EXTERNAL_VERSION, closeDialog)}
					>
						{#snippet titleRight()}
							<span
								class="ml-0.5 inline-flex translate-0.5 items-center gap-1 text-gray-300"
							>
								<ExternalLinkIcon size={16} class="stroke-2" />
								{#if currentVersionProbe.isExternal}
									<Badge variant="secondary" class="mr-3 text-[10px] font-medium">
										{m.Wallet_version_current()}
									</Badge>
								{/if}
							</span>
						{/snippet}
					</ItemCard>

					<WithEmptyState
						items={foundVersions}
						emptyText={m.No_wallet_versions_found()}
						containerClass="[&>div>div]:p-0!"
					>
						{#snippet item({ item })}
							<ItemCard
								title={item.tag}
								onClick={() => applyVersionAndClose(item, closeDialog)}
							>
								{#snippet titleRight()}
									<span
										class="ml-0.5 inline-flex translate-0.5 items-center gap-1 text-gray-300"
									>
										{#if item.android_installer}
											<AndroidLogo size={16} />
										{/if}
										{#if item.ios_installer}
											<AppleLogo size={16} />
										{/if}
										{#if isCurrentWalletVersionRow(item)}
											<Badge
												variant="secondary"
												class="mr-3 text-[10px] font-medium"
											>
												{m.Wallet_version_current()}
											</Badge>
										{/if}
									</span>
								{/snippet}
							</ItemCard>
						{/snippet}
					</WithEmptyState>
				{/if}
			</div>
		{/if}
	{/snippet}
</Dialog>
