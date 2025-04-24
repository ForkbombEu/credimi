<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageGrid from '$lib/layout/pageGrid.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';
	import WalletCard from '$lib/layout/walletCard.svelte';
	import CollectionManager from '@/collections-components/manager/collectionManager.svelte';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { localizeHref, m } from '@/i18n';
	import { pb } from '@/pocketbase';
	import type { Collections } from '@/pocketbase/types';
	import { truncate } from 'lodash';

	/**
	 * This type is needed as the MarketplaceItem type coming from codegen is not good.
	 * Since `marketplace_items` is a view collection, that merges multiple collections,
	 * pocketbase says that each field is of type `json` and not the actual type.
	 */
	type MarketplaceItem = {
		collectionId: string;
		collectionName: string;
		id: string;
		type: Collections;
		name: string;
		description: string | null;
		avatar: string | null;
		avatar_url: string | null;
	};

	const displayData: Partial<
		Record<Collections, { label: string; bgClass: string; textClass: string }>
	> = {
		wallets: { label: m.Wallet(), bgClass: 'bg-blue-500', textClass: 'text-blue-500' },
		verifiers: { label: m.Verifier(), bgClass: 'bg-green-500', textClass: 'text-green-500' },
		credential_issuers: {
			label: m.Credential_issuer(),
			bgClass: 'bg-yellow-500',
			textClass: 'text-yellow-500'
		}
	};
</script>

<CollectionManager collection="marketplace_items">
	{#snippet top({ Search })}
		<PageTop>
			<T tag="h1">{m.Marketplace()}</T>
			<Search
				class="border-primary bg-secondary pur                                                                                                                                                                                                                                                                                                                   "
			/>
		</PageTop>
	{/snippet}

	{#snippet contentWrapper(children)}
		<PageContent class="bg-secondary grow">
			{@render children()}
		</PageContent>
	{/snippet}

	{#snippet records({ records })}
		<PageGrid>
			{#each records as record}
				{@const item = record as MarketplaceItem}
				{@const href = localizeHref(`/marketplace/${item.type}/${item.id}`)}
				{@const logo = item.avatar ? pb.files.getURL(item, item.avatar) : item.avatar_url}
				{@const display = displayData[item.type]}

				<a
					{href}
					class="border-primary bg-card text-card-foreground ring-primary flex flex-col justify-between gap-4 rounded-lg border p-6 shadow-sm transition-all hover:-translate-y-2 hover:ring-2"
				>
					<div class="space-y-1">
						<T class="overflow-hidden text-ellipsis font-semibold">{item.name}</T>
						<div class="flex items-center gap-1">
							<div class="{display?.bgClass} size-1.5 rounded-full"></div>
							<T class="{display?.textClass} text-sm">{display?.label}</T>
						</div>
						{#if item.description}
							<T class="text-muted-foreground pt-1 text-sm">
								{truncate(item.description, { length: 100 })}
							</T>
						{/if}
					</div>

					<div class="flex items-end justify-between gap-2">
						<T class="text-muted-foreground text-xs">Last check: yyyy-mm-dd</T>
						{#if logo}
							<Avatar
								src={logo}
								class="size-14 !rounded-sm border"
								hideIfLoadingError
							/>
						{/if}
					</div>

					<!-- <div
						class="text-muted-foreground flex flex-col items-start gap-2 overflow-hidden"
					>
						{#if String.isNonEmpty(service.url)}
							<T tag="small">{service.url}</T>
						{/if}
						{#if String.isNonEmpty(service.homepage_url)}
							<T tag="small">{service.homepage_url}</T>
						{/if}
						{#if String.isNonEmpty(service.repo_url)}
							<T tag="small">{service.repo_url}</T>
						{/if}
					</div> -->
				</a>
			{/each}
		</PageGrid>
	{/snippet}
</CollectionManager>
