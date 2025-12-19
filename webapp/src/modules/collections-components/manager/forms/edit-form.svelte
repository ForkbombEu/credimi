<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="C extends CollectionName">
	import { merge } from 'lodash';

	import type { CollectionFormOptions } from '@/collections-components/form/collectionFormTypes';
	import type { CollectionName } from '@/pocketbase/collections-models';

	import { CollectionForm } from '@/collections-components';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import { FormError, SubmitButton } from '@/forms';
	import { m } from '@/i18n';

	import type { RecordEditProps } from '../record-actions/types';

	import { getCollectionManagerContext } from '../collectionManagerContext';

	//

	const { manager, formsOptions, formRefineSchema, editForm } = $derived(
		getCollectionManagerContext()
	);

	const props = $derived(manager.editFormProps as RecordEditProps<C> | undefined);

	const defaultFormOptions: CollectionFormOptions<C> = {
		uiOptions: { showToastOnSuccess: true }
	};
	const options = $derived(merge(defaultFormOptions, formsOptions.base, formsOptions.edit));

	const sheetTitle = $derived(props?.formTitle ?? m.Edit_record());
</script>

<Sheet title={sheetTitle} class="pb-0" hideTrigger bind:open={manager.isEditFormOpen}>
	{#snippet content()}
		{@const record = props?.record}
		{#if record}
			{#if editForm}
				{@render editForm({
					record: props?.record as never,
					closeSheet: () => manager.closeEditForm()
				})}
			{:else}
				<CollectionForm
					collection={manager.collection}
					recordId={record.id}
					initialData={record as unknown as undefined}
					refineSchema={formRefineSchema}
					{...options}
					onSuccess={(record) => {
						props?.onSuccess?.(record, 'edit');
						manager.loadRecords();
						manager.closeEditForm();
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
							{#if props?.buttonText}
								{@render props.buttonText()}
							{:else}
								{m.Save()}
							{/if}
						</SubmitButton>
					</div>
				</CollectionForm>
			{/if}
		{/if}
	{/snippet}
</Sheet>
