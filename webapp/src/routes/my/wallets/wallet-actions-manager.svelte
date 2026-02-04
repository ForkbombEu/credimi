<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { yaml } from '@codemirror/lang-yaml';
	import { UploadIcon } from '@lucide/svelte';
	import WalletActionTags from '$lib/components/wallet-action-tags.svelte';
	import DashboardCardManagerTop from '$lib/layout/dashboard-card-manager-top.svelte';
	import DashboardCardManagerUI from '$lib/layout/dashboard-card-manager-ui.svelte';
	import { yamlStringSchema } from '$lib/utils';
	import { z } from 'zod/v3';

	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes';
	import type { OrganizationsResponse, WalletsResponse } from '@/pocketbase/types';

	import { CollectionManager } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import { CodeEditorField } from '@/forms/fields';
	import { m } from '@/i18n';
	import { readFileAsString, startFileUpload } from '@/utils/files';

	//

	type Props = {
		wallet: WalletsResponse;
		organization: OrganizationsResponse;
	};

	let { wallet, organization }: Props = $props();

	//
</script>

<CollectionManager
	collection="wallet_actions"
	hide={['empty_state']}
	queryOptions={{ filter: `wallet.id = '${wallet.id}' && owner.id = '${organization.id}'` }}
	formRefineSchema={(schema) =>
		schema.extend({
			code: yamlStringSchema as unknown as z.ZodString
		})}
	formFieldsOptions={{
		exclude: ['owner', 'canonified_name', 'published'],
		hide: { wallet: wallet.id },
		snippets: { code: codeField },
		placeholders: {
			name: m.e_g_Get_Credential(),
			tags: 'e.g. v.0.01, Above 18 credential'
		},
		descriptions: {
			tags: m.separate_items_by_commas()
		}
	}}
>
	{#snippet top()}
		<DashboardCardManagerTop
			label={m.Wallet_actions()}
			buttonText={m.Add_new_action()}
			recordCreateOptions={{
				formTitle: `${m.Wallet()}: ${wallet.name} — ${m.Add_new_action()}`
			}}
		/>
	{/snippet}

	{#snippet records({ records })}
		<DashboardCardManagerUI {records} nameField="name">
			{#snippet actions({ record })}
				<WalletActionTags action={record} containerClass="justify-end" />
			{/snippet}
		</DashboardCardManagerUI>
	{/snippet}
</CollectionManager>

<!--  -->

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
