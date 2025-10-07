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
	import { FormError, SubmitButton } from '@/forms';
	import { m } from '@/i18n';

	import type { RecordCreateEditProps } from './types';

	import { getCollectionManagerContext } from '../collectionManagerContext';

	//

	type Props = RecordCreateEditProps<C> & {
		record: CollectionResponses[C];
	};

	const { formTitle, onSuccess = () => {}, buttonText, button, record }: Props = $props();

	//

	const { manager, formsOptions, formRefineSchema } = $derived(getCollectionManagerContext());

	const defaultFormOptions: CollectionFormOptions<C> = {
		uiOptions: { showToastOnSuccess: true }
	};
	const options = $derived(merge(defaultFormOptions, formsOptions.base, formsOptions.edit));

	const sheetTitle = $derived(formTitle ?? m.Edit_record());
</script>

<Sheet title={sheetTitle} class="pb-0">
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
			initialData={record as unknown as undefined}
			refineSchema={formRefineSchema}
			{...options}
			onSuccess={(record) => {
				closeSheet();
				manager.loadRecords();
				onSuccess(record, 'create');
			}}
			uiOptions={{
				hide: ['submit_button', 'error']
			}}
		>
			<FormError />
			<div
				class="sticky bottom-0 -mx-6 -mt-6 flex justify-end border-t bg-white/70 px-6 py-2 backdrop-blur-sm"
			>
				<SubmitButton>
					{#if buttonText}
						{@render buttonText?.()}
					{:else}
						{m.Save()}
					{/if}
				</SubmitButton>
			</div>
		</CollectionForm>
	{/snippet}
</Sheet>
