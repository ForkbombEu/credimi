<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CollectionManager } from '@/collections-components';
	import Card from '@/components/ui-custom/card.svelte';
	import type { WalletsResponse } from '@/pocketbase/types';
	import type { ConformanceCheck } from './wallet-form-checks-table.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import A from '@/components/ui-custom/a.svelte';
	import { Badge } from '@/components/ui/badge';
	import RenderMd from '@/components/ui-custom/renderMD.svelte';
	import { RecordDelete } from '@/collections-components/manager';
	import Button from '@/components/ui-custom/button.svelte';
	import { Separator } from '@/components/ui/separator';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import WalletForm from './wallet-form.svelte';
	import { Pencil, Plus } from 'lucide-svelte';
	import { m } from '@/i18n';
	import SwitchWithIcons from '@/components/ui-custom/switch-with-icons.svelte';
	import { Eye, EyeOff } from 'lucide-svelte';
	import { pb } from '@/pocketbase';
	import type { WorkflowExecution } from '@forkbombeu/temporal-ui/dist/types/workflows';

	//

	type Props = {
		organizationId?: string;
		workflows?: WorkflowExecution[];
		id?: string;
	};

	let { organizationId, id }: Props = $props();

	//

	function updatePublished(
		recordId: string,
		published: boolean,
		onSuccess: () => void
	) {
		pb.collection('wallets')
			.update(recordId, {
				published
			})
			.then(() => {
				onSuccess();
			});
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
	{#snippet top({ Header })}
		<Header title="Wallets" {id}>
			{#snippet right()}
				{@render WalletFormSnippet()}
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

		<div class="space-y-4">
			<div class="flex flex-row items-start justify-between gap-4">
				<div>
					<div class="flex items-center gap-2">
						<T class="font-bold">
							{#if !wallet.published}
								{wallet.name}
							{:else}
								<A href="/apps/{wallet.id}">{wallet.name}</A>
							{/if}
						</T>
					</div>
					<T class="mt-1 text-xs text-gray-400">
						<RenderMd content={wallet.description} />
					</T>
				</div>

				<div class="flex items-center gap-1">
					<SwitchWithIcons
						offIcon={EyeOff}
						onIcon={Eye}
						size="md"
						checked={wallet.published}
						onCheckedChange={() =>
							updatePublished(
								wallet.id,
								!wallet.published,
								onEditSuccess
							)}
					/>

					{@render UpdateWalletFormSnippet(wallet.id, wallet, onEditSuccess)}

					<RecordDelete record={wallet}>
						{#snippet button({ triggerAttributes, icon: Icon })}
							<Button variant="outline" size="sm" class="p-2" {...triggerAttributes}>
								<Icon />
							</Button>
						{/snippet}
					</RecordDelete>
				</div>
			</div>

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

{#snippet WalletFormSnippet()}
	<Sheet>
		{#snippet trigger({ sheetTriggerAttributes })}
			<!-- {#if workflows?.length === 0}
				<Button disabled variant="outline" class="text-wrap text-xs">
					{m.Before_adding_a_new_wallet_you_need_to_start_a_conformance_check()}
				</Button>
			{:else} -->
			<Button {...sheetTriggerAttributes}>
				<Plus />{m.Add_new_wallet()}
			</Button>
			<!-- {/if} -->
		{/snippet}

		{#snippet content({ closeSheet })}
			<div class="space-y-6">
				<T tag="h3">Add a new wallet</T>
				<WalletForm onSuccess={closeSheet} />
			</div>
		{/snippet}
	</Sheet>
{/snippet}

{#snippet UpdateWalletFormSnippet(
	walletId: string,
	initialData: Partial<WalletsResponse>,
	onEditSuccess: () => void
)}
	<Sheet>
		{#snippet trigger({ sheetTriggerAttributes })}
			<Button variant="outline" size="sm" class="p-2" {...sheetTriggerAttributes}>
				<Pencil />
			</Button>
		{/snippet}

		{#snippet content({ closeSheet })}
			<div class="space-y-6">
				<T tag="h3">Add a new wallet</T>
				<WalletForm
					{walletId}
					{initialData}
					onSuccess={() => {
						onEditSuccess();
						closeSheet();
					}}
				/>
			</div>
		{/snippet}
	</Sheet>
{/snippet}
