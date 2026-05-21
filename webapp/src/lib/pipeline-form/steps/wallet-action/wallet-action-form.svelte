<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { userOrganization } from '$lib/app-state';
	import { yamlStringSchema } from '$lib/utils';

	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes';
	import type { WalletActionsResponse } from '@/pocketbase/types';
	import type { CollectionZodSchema } from '@/pocketbase/zod-schema';

	import { CollectionForm } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { CodeEditorField } from '@/forms/fields';
	import { m } from '@/i18n';

	//

	type Props = {
		walletAction: WalletActionsResponse;
	};

	let { walletAction }: Props = $props();

	const currentUserIsNotOwner = $derived(walletAction.owner !== userOrganization.current?.id);
</script>

<Sheet title={m.Edit_action()}>
	{#snippet trigger({ sheetTriggerAttributes })}
		<Tooltip disabled={!currentUserIsNotOwner}>
			<Button
				{...sheetTriggerAttributes}
				class="h-fit! shrink-0 p-0! disabled:text-muted-foreground disabled:hover:cursor-not-allowed"
				variant="link"
				disabled={currentUserIsNotOwner}
			>
				{m.Edit_action()}
			</Button>

			{#snippet content()}
				{m.Editing_is_disabled_because_you_are_not_the_owner_of_the_action()}
			{/snippet}
		</Tooltip>
	{/snippet}

	{#snippet content({ closeSheet })}
		<CollectionForm
			collection="wallet_actions"
			recordId={walletAction.id}
			initialData={{ code: walletAction.code }}
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
