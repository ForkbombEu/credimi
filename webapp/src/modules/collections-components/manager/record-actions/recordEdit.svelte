<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="C extends CollectionName">
	import { merge } from 'lodash';
	import Pencil from 'lucide-svelte/icons/pencil';

	import type { CollectionFormOptions } from '@/collections-components/form/collectionFormTypes';
	import type { CollectionName } from '@/pocketbase/collections-models';
	import type { CollectionResponses } from '@/pocketbase/types';

	import { CollectionForm } from '@/collections-components';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import { m } from '@/i18n';

	import type { RecordCreateEditProps } from './types';

	import { getCollectionManagerContext } from '../collectionManagerContext';

	//

	type Props = RecordCreateEditProps<C> & {
		record: CollectionResponses[C];
	};

	const { formTitle, onSuccess = () => {}, buttonText, button, record }: Props = $props();

	//

	const { manager, formsOptions } = $derived(getCollectionManagerContext());

	const defaultFormOptions: CollectionFormOptions<C> = {
		uiOptions: { showToastOnSuccess: true }
	};
	const options = $derived(merge(defaultFormOptions, formsOptions.base, formsOptions.edit));

	const sheetTitle = $derived(formTitle ?? m.Edit_record());
</script>

<Sheet title={sheetTitle}>
	{#snippet trigger({ sheetTriggerAttributes, openSheet })}
		{#if button}
			{@render button({
				triggerAttributes: sheetTriggerAttributes,
				icon: Pencil,
				openForm: openSheet
			})}
		{:else}
			<IconButton variant="outline" icon={Pencil} {...sheetTriggerAttributes} />
		{/if}
	{/snippet}

	{#snippet content({ closeSheet })}
		<CollectionForm
			collection={manager.collection}
			recordId={record.id}
			initialData={record}
			{...options}
			onSuccess={(record) => {
				closeSheet();
				manager.loadRecords();
				onSuccess(record, 'create');
			}}
		>
			{#snippet submitButtonContent()}
				{#if buttonText}
					{@render buttonText?.()}
				{:else}
					{m.Edit_record()}
				{/if}
			{/snippet}
		</CollectionForm>
	{/snippet}
</Sheet>
