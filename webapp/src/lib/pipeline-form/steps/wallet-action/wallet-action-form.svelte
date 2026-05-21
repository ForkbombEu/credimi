<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { yamlStringSchema } from '$lib/utils';

	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes';
	import type { WalletActionsResponse } from '@/pocketbase/types';
	import type { CollectionZodSchema } from '@/pocketbase/zod-schema';

	import { CollectionForm } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import { CodeEditorField } from '@/forms/fields';
	import { m } from '@/i18n';

	//

	type Props = {
		walletAction: WalletActionsResponse;
	};

	let { walletAction }: Props = $props();
</script>

<Sheet title={m.Edit_action()}>
	{#snippet trigger({ sheetTriggerAttributes })}
		<Button {...sheetTriggerAttributes} class="h-fit! shrink-0 p-0!" variant="link">
			{m.Edit_action()}
		</Button>
	{/snippet}

	{#snippet content({ closeSheet })}
		<CollectionForm
			collection="wallet_actions"
			recordId={walletAction.id}
			initialData={walletAction}
			fieldsOptions={{
				include: ['code'],
				snippets: { code }
			}}
			refineSchema={(schema) => {
				return schema.extend({
					code: yamlStringSchema
				}) as unknown as CollectionZodSchema<'wallet_actions'>;
			}}
			onSuccess={() => {
				closeSheet();
			}}
		/>
	{/snippet}
</Sheet>

{#snippet code({ form }: FieldSnippetOptions<'wallet_actions'>)}
	<CodeEditorField
		{form}
		name="code"
		options={{ lang: 'yaml', minHeight: 300, maxHeight: 700 }}
	/>
{/snippet}
