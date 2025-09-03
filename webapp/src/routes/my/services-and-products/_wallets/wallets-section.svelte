<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { WorkflowExecution } from '@forkbombeu/temporal-ui/dist/types/workflows';

	import { ChevronDown, ChevronUp, Eye, EyeOff } from 'lucide-svelte';

	import type { WalletsResponse } from '@/pocketbase/types';

	import { CollectionManager } from '@/collections-components';
	import { RecordDelete } from '@/collections-components/manager';
	import A from '@/components/ui-custom/a.svelte';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import Card from '@/components/ui-custom/card.svelte';
	import RenderMd from '@/components/ui-custom/renderMD.svelte';
	import SwitchWithIcons from '@/components/ui-custom/switch-with-icons.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { Separator } from '@/components/ui/separator';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import type { ConformanceCheck } from './wallet-form-checks-table.svelte';

	import WalletFormSheet from './wallet-form-sheet.svelte';

	//

	type Props = {
		organizationId?: string;
		workflows?: WorkflowExecution[];
		id?: string;
	};

	let { organizationId, id }: Props = $props();

	//

	let expandedDescriptions = $state(new Set<string>());

	//

	function updatePublished(recordId: string, published: boolean, onSuccess: () => void) {
		pb.collection('wallets')
			.update(recordId, {
				published
			})
			.then(() => {
				onSuccess();
			});
	}

	function toggleDescriptionExpansion(walletId: string) {
		if (expandedDescriptions.has(walletId)) {
			expandedDescriptions.delete(walletId);
		} else {
			expandedDescriptions.add(walletId);
		}
		expandedDescriptions = new Set(expandedDescriptions);
	}
</script>

<CollectionManager
	collection="wallets"
	queryOptions={{
		filter: `owner.id = '${organizationId}'`,
		sort: ['created', 'DESC']
	}}
	editFormFieldsOptions={{ exclude: ['owner', 'published'] }}
>
	{#snippet top({ Header, reloadRecords })}
		<Header title="Wallets" {id}>
			{#snippet right()}
				<WalletFormSheet onEditSuccess={reloadRecords} />
			{/snippet}
		</Header>
	{/snippet}

	{#snippet records({ records, reloadRecords })}
		<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
			{#each records as record (record.id)}
				{@render WalletCard(record, reloadRecords)}
			{/each}
		</div>
	{/snippet}
</CollectionManager>

{#snippet WalletCard(wallet: WalletsResponse, onEditSuccess: () => void)}
	<Card class="bg-background">
		{@const conformanceChecks = wallet.conformance_checks as
			| ConformanceCheck[]
			| null
			| undefined}
		{@const avatarSrc = wallet.logo ? pb.files.getURL(wallet, wallet.logo) : wallet.logo_url}

		<div class="space-y-4">
			<div class="flex flex-row items-start justify-between gap-4">
				<Avatar src={avatarSrc} fallback={wallet.name} class="rounded-sm border" />
				<div class="flex-1">
					<div class="flex items-center gap-2">
						<T class="font-bold">
							{#if !wallet.published}
								{wallet.name}
							{:else}
								<A href="/marketplace/wallets/{wallet.id}">{wallet.name}</A>
							{/if}
						</T>
					</div>
					{#if wallet.appstore_url}
						<T class="text-xs text-gray-400">{wallet.appstore_url}</T>
					{/if}
				</div>

				<div class="flex items-center gap-1">
					<SwitchWithIcons
						offIcon={EyeOff}
						onIcon={Eye}
						size="md"
						checked={wallet.published}
						onCheckedChange={() =>
							updatePublished(wallet.id, !wallet.published, onEditSuccess)}
					/>
					<WalletFormSheet walletId={wallet.id} initialData={wallet} {onEditSuccess} />
					<RecordDelete record={wallet}>
						{#snippet button({ triggerAttributes, icon: Icon })}
							<Button variant="outline" size="sm" class="p-2" {...triggerAttributes}>
								<Icon />
							</Button>
						{/snippet}
					</RecordDelete>
				</div>
			</div>

			{#if wallet.description}
				<Separator />
				{@const isExpanded = expandedDescriptions.has(wallet.id)}
				{@const descriptionText = wallet.description.replace(/<[^>]*>/g, '').trim()}
				{@const needsExpansion = descriptionText.length > 100}
				<div class="mt-1 text-xs text-gray-400">
					<div
						class="transition-all duration-200 ease-in-out {isExpanded
							? ''
							: 'line-clamp-2'}"
					>
						<RenderMd content={wallet.description} />
					</div>

					{#if needsExpansion}
						<button
							class="text-primary mt-1 flex items-center gap-1 text-xs transition-colors duration-150 hover:underline"
							onclick={() => toggleDescriptionExpansion(wallet.id)}
							type="button"
						>
							{#if isExpanded}
								{m.Show_less()}
								<ChevronUp class="h-3 w-3" />
							{:else}
								{m.Show_more()}
								<ChevronDown class="h-3 w-3" />
							{/if}
						</button>
					{/if}
				</div>
			{/if}

			<Separator />

			<div class="flex flex-wrap gap-2">
				{#if conformanceChecks && conformanceChecks.length > 0}
					{#each conformanceChecks as check}
						<Badge variant={check.status === 'success' ? 'secondary' : 'destructive'}>
							{check.test}
						</Badge>
					{/each}
				{:else}
					<T class="text-gray-300">
						{m.No_conformance_checks_available()}
					</T>
				{/if}
			</div>
		</div>
	</Card>
{/snippet}
