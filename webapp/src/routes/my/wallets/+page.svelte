<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ConformanceCheck } from '$lib/types/checks';

	import { yaml } from '@codemirror/lang-yaml';
	import DashboardCardManagerTop from '$lib/layout/dashboard-card-manager-top.svelte';
	import DashboardCardManagerUI from '$lib/layout/dashboard-card-manager-ui.svelte';
	import DashboardCard from '$lib/layout/dashboard-card.svelte';
	import { yamlStringSchema } from '$lib/utils';
	import { EyeIcon, UploadIcon } from 'lucide-svelte';
	import { z } from 'zod';

	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes';
	import type { WalletsResponse } from '@/pocketbase/types';

	import { CollectionManager } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { Separator } from '@/components/ui/separator';
	import { CodeEditorField } from '@/forms/fields';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';
	import { readFileAsString, startFileUpload } from '@/utils/files';

	import { setDashboardNavbar } from '../+layout@.svelte';
	import WalletFormSheet from './wallet-form-sheet.svelte';
	import WalletForm from './wallet-form.svelte';

	//

	let { data } = $props();
	let { organization } = $derived(data);

	setDashboardNavbar({ title: 'Wallets', right: navbarRight });
</script>

<CollectionManager
	collection="wallets"
	queryOptions={{
		filter: `owner.id = '${organization.id}'`,
		sort: ['created', 'DESC']
	}}
>
	{#snippet editForm({ record: wallet, closeSheet })}
		<WalletForm walletId={wallet.id} initialData={wallet} onSuccess={() => closeSheet()} />
	{/snippet}

	{#snippet records({ records })}
		<div class="space-y-6">
			{#each records as record (record.id)}
				<DashboardCard
					{record}
					avatar={(w) => (w.logo ? pb.files.getURL(w, w.logo) : w.logo_url)}
					path={[organization.canonified_name, record.canonified_name]}
				>
					{#snippet content()}
						{@const conformanceChecks = record.conformance_checks as
							| ConformanceCheck[]
							| null
							| undefined}
						<div class="flex flex-wrap gap-2">
							{#if conformanceChecks && conformanceChecks.length > 0}
								{#each conformanceChecks as check (check)}
									<Badge
										variant={check.status === 'success'
											? 'secondary'
											: 'destructive'}
									>
										{check.test}
									</Badge>
								{/each}
							{:else}
								<T class="text-sm text-gray-300">
									{m.No_conformance_checks_available()}
								</T>
							{/if}
						</div>

						<Separator />

						{@render walletVersionsManager({
							wallet: record,
							organizationId: organization.id
						})}

						<Separator />

						{@render walletActionsManager({ wallet: record, ownerId: record.owner })}
					{/snippet}
				</DashboardCard>
			{/each}
		</div>
	{/snippet}
</CollectionManager>

<!--  -->

{#snippet navbarRight()}
	<WalletFormSheet />
{/snippet}

{#snippet walletVersionsManager(props: { wallet: WalletsResponse; organizationId: string })}
	{@const wallet = props.wallet}
	<CollectionManager
		collection="wallet_versions"
		queryOptions={{
			filter: `wallet = '${wallet.id}' && owner.id = '${props.organizationId}'`
		}}
		hide={['empty_state']}
		formFieldsOptions={{
			exclude: ['owner'],
			hide: { wallet: wallet.id },
			placeholders: {
				android_installer: m.Upload_a_new_file(),
				ios_installer: m.Upload_a_new_file(),
				tag: 'e.g. v1.0.0'
			},
			labels: {
				tag: m.Tag(),
				android_installer: m.Android_installer(),
				ios_installer: m.iOS_installer()
			}
		}}
	>
		{#snippet top()}
			<DashboardCardManagerTop
				label={m.Wallet_versions()}
				buttonText={m.Add_new_version()}
				recordCreateOptions={{
					uiOptions: { hideRequiredIndicator: true },
					formTitle: `${m.Wallet()}: ${wallet.name} — ${m.Add_new_version()}`
				}}
			/>
		{/snippet}

		{#snippet records({ records })}
			<DashboardCardManagerUI
				{records}
				nameField="tag"
				hideClone
				path={(r) => [
					organization.canonified_name,
					props.wallet.canonified_name,
					r.canonified_tag
				]}
			>
				{#snippet actions({ record })}
					<div class="flex items-center gap-1">
						{#if record.ios_installer}
							<Badge>iOS</Badge>
						{/if}
						{#if record.android_installer}
							<Badge>Android</Badge>
						{/if}
					</div>
				{/snippet}
			</DashboardCardManagerUI>
		{/snippet}
	</CollectionManager>
{/snippet}

{#snippet walletActionsManager(props: { wallet: WalletsResponse; ownerId: string })}
	<CollectionManager
		collection="wallet_actions"
		hide={['empty_state']}
		queryOptions={{
			filter: `wallet.id = '${props.wallet.id}'`
		}}
		formRefineSchema={(schema) =>
			schema.extend({
				code: yamlStringSchema as unknown as z.ZodString
			})}
		formFieldsOptions={{
			exclude: ['owner', 'canonified_name', 'result'],
			hide: { wallet: props.wallet.id, owner: props.ownerId },
			snippets: { code: codeField },
			placeholders: {
				name: m.e_g_Get_Credential()
			}
		}}
	>
		{#snippet top()}
			<DashboardCardManagerTop
				label={m.Wallet_actions()}
				buttonText={m.Add_new_action()}
				recordCreateOptions={{
					formTitle: `${m.Wallet()}: ${props.wallet.name} — ${m.Add_new_action()}`
				}}
			/>
		{/snippet}

		{#snippet records({ records })}
			<DashboardCardManagerUI
				{records}
				nameField="name"
				path={(r) => [
					organization.canonified_name,
					props.wallet.canonified_name,
					r.canonified_name
				]}
			>
				{#snippet actions({ record })}
					{#if record.result}
						<Button
							size="sm"
							variant="outline"
							class="h-8 border border-blue-500"
							href={pb.files.getURL(record, record.result)}
							target="_blank"
						>
							<EyeIcon />
							View result
						</Button>
					{/if}
				{/snippet}
			</DashboardCardManagerUI>
		{/snippet}
	</CollectionManager>
{/snippet}

{#snippet codeField(options: FieldSnippetOptions<'wallet_actions'>)}
	<CodeEditorField
		form={options.form}
		name={options.field}
		options={{ lang: yaml(), minHeight: 300, maxHeight: 700, labelRight }}
	/>

	{#snippet labelRight()}
		<Button
			variant="secondary"
			size="sm"
			onclick={() =>
				startFileUpload({
					onLoad: async (file) => {
						const code = await readFileAsString(file);
						options.form.form.update((data) => ({
							...data,
							code
						}));
					}
				})}
		>
			<UploadIcon />
			{m.Upload_yaml()}
		</Button>
	{/snippet}
{/snippet}
