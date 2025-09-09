<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { WorkflowExecution } from '@forkbombeu/temporal-ui/dist/types/workflows';

	import { yaml } from '@codemirror/lang-yaml';
	import { ChevronDown, ChevronUp, Eye, EyeOff, UploadIcon } from 'lucide-svelte';

	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes';
	import type { IconComponent } from '@/components/types';
	import type { WalletsResponse } from '@/pocketbase/types';

	import { CollectionManager } from '@/collections-components';
	import { RecordCreate, RecordDelete, RecordEdit } from '@/collections-components/manager';
	import A from '@/components/ui-custom/a.svelte';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import Card from '@/components/ui-custom/card.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import RenderMd from '@/components/ui-custom/renderMD.svelte';
	import SwitchWithIcons from '@/components/ui-custom/switch-with-icons.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { Separator } from '@/components/ui/separator';
	import { CodeEditorField } from '@/forms/fields';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';
	import { readFileAsString, startFileUpload } from '@/utils/files';

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
							<Button
								variant="outline"
								size="icon"
								class="p-2"
								{...triggerAttributes}
							>
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

			<Separator />

			{@render walletVersionsManager({
				wallet,
				organizationId: organizationId ?? ''
			})}

			<Separator />

			{@render walletActionsManager({ wallet, ownerId: wallet.owner })}
		</div>
	</Card>
{/snippet}

<!-- Versions -->

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
		{#snippet records({ records })}
			<div>
				<div class="mb-2 flex items-center justify-between">
					<T class="text-sm font-semibold">{m.Wallet_versions()}</T>
					{@render createVersion(wallet)}
				</div>
				<ul class="space-y-2">
					{#each records as record}
						<li
							class="bg-muted flex items-center justify-between rounded-md p-2 pl-3 pr-2"
						>
							{record.tag}
							<div class="flex items-center gap-2">
								<div class="flex items-center gap-1">
									{#if record.ios_installer}
										<Badge>iOS</Badge>
									{/if}
									{#if record.android_installer}
										<Badge>Android</Badge>
									{/if}
								</div>

								<div>
									<RecordEdit
										{record}
										uiOptions={{ hideRequiredIndicator: true }}
										formTitle={`${m.Wallet()}: ${wallet.name} — ${m.Edit_version()}: ${record.tag}`}
									>
										{#snippet button({ triggerAttributes, icon })}
											<IconButton
												variant="outline"
												size="sm"
												{icon}
												{...triggerAttributes}
											/>
										{/snippet}
									</RecordEdit>
									{@render recordDelete(record)}
								</div>
							</div>
						</li>
					{/each}
				</ul>
			</div>
		{/snippet}

		{#snippet emptyState()}
			<div class="flex items-center justify-between">
				<T class="text-gray-300">
					{m.No_wallet_versions_available()}
				</T>
				{@render createVersion(wallet)}
			</div>
		{/snippet}
	</CollectionManager>
{/snippet}

{#snippet createVersion(wallet: WalletsResponse)}
	<RecordCreate
		uiOptions={{ hideRequiredIndicator: true }}
		formTitle={`${m.Wallet()}: ${wallet.name} — ${m.Add_new_version()}`}
	>
		{#snippet button({ triggerAttributes, icon })}
			{@render blueButton({ triggerAttributes, icon, text: m.Add_new_version() })}
		{/snippet}
	</RecordCreate>
{/snippet}

<!-- Actions -->

{#snippet walletActionsManager(props: { wallet: WalletsResponse; ownerId: string })}
	<CollectionManager
		collection="wallet_actions"
		hide={['empty_state', 'pagination']}
		queryOptions={{
			filter: `wallet.id = '${props.wallet.id}'`
		}}
		formFieldsOptions={{
			exclude: ['owner'],
			hide: { wallet: props.wallet.id, owner: props.ownerId },
			snippets: { code: codeField },
			labels: {
				uid: 'UID'
			},
			descriptions: {
				uid: m.Only_lowercase_letters_and_underscores_are_allowed()
			},
			placeholders: {
				name: m.e_g_Get_Credential(),
				uid: m.e_g_get_credential_uid()
			}
		}}
	>
		{#snippet records({ records })}
			<div>
				<div class="mb-2 flex items-center justify-between">
					<T class="text-sm font-semibold">{m.Wallet_actions()}</T>
					<RecordCreate
						formTitle={`${m.Wallet()}: ${props.wallet.name} — ${m.Add_new_action()}`}
					>
						{#snippet button({ triggerAttributes, icon })}
							{@render blueButton({
								triggerAttributes,
								icon,
								text: m.Add_new_action()
							})}
						{/snippet}
					</RecordCreate>
				</div>
				<ul class="space-y-2">
					{#each records as record}
						<li
							class="bg-muted flex items-center justify-between rounded-md p-2 pl-3 pr-2"
						>
							{record.name}
							<div class="flex items-center gap-1">
								<RecordEdit
									{record}
									formTitle={`${m.Wallet()}: ${props.wallet.name} — ${m.Edit_action()}: ${record.name}`}
								>
									{#snippet button({ triggerAttributes, icon })}
										<IconButton
											size="sm"
											variant="outline"
											{icon}
											{...triggerAttributes}
										/>
									{/snippet}
								</RecordEdit>
								{@render recordDelete(record)}
							</div>
						</li>
					{/each}
				</ul>
			</div>
		{/snippet}

		{#snippet emptyState()}
			<div class="flex items-center justify-between gap-2">
				<T class="text-gray-300">
					{m.No_actions_available()}
				</T>
				<RecordCreate>
					{#snippet button({ triggerAttributes, icon })}
						{@render blueButton({
							triggerAttributes,
							icon,
							text: m.Add_first_action()
						})}
					{/snippet}
				</RecordCreate>
			</div>
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

{#snippet blueButton(props: { triggerAttributes: object; icon: IconComponent; text: string })}
	<Button
		variant="link"
		size="sm"
		class="h-8 gap-1 px-2 text-blue-600 hover:cursor-pointer hover:bg-blue-50 hover:no-underline"
		{...props.triggerAttributes}
	>
		<Icon src={props.icon} />
		{props.text}
	</Button>
{/snippet}

<!-- eslint-disable-next-line @typescript-eslint/no-explicit-any -->
{#snippet recordDelete(record: any)}
	<RecordDelete {record}>
		{#snippet button({ triggerAttributes, icon })}
			<IconButton variant="outline" size="sm" {icon} {...triggerAttributes} />
		{/snippet}
	</RecordDelete>
{/snippet}
