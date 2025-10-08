<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="C extends CollectionName">
	import { merge } from 'lodash';
	import { Plus } from 'lucide-svelte';

	import type { CollectionName } from '@/pocketbase/collections-models';

	import { CollectionForm } from '@/collections-components';
	import { type CollectionFormOptions } from '@/collections-components/form/collectionFormTypes';
	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import { FormError, SubmitButton } from '@/forms';
	import { m } from '@/i18n';

	import type { RecordCreateEditProps } from './types';

	import { getCollectionManagerContext } from '../collectionManagerContext';

	//

	const {
		formTitle,
		onSuccess = () => {},
		buttonText,
		button,
		form
	}: RecordCreateEditProps<C> = $props();

	const { manager, formsOptions, formRefineSchema } = $derived(getCollectionManagerContext());

	const defaultFormOptions: CollectionFormOptions<C> = {
		uiOptions: { showToastOnSuccess: true }
	};
	const options = $derived(merge(defaultFormOptions, formsOptions.base, formsOptions.edit));

	const sheetTitle = $derived(formTitle ?? m.Create_record());
</script>

<Sheet title={sheetTitle} class="pb-0">
	{#snippet trigger({ sheetTriggerAttributes, openSheet })}
		{#if button}
			{@render button({
				triggerAttributes: sheetTriggerAttributes,
				icon: Plus,
				openForm: openSheet
			})}
		{:else}
			<Button {...sheetTriggerAttributes} class="shrink-0">
				<Icon src={Plus} />
				{@render SubmitButtonText()}
			</Button>
		{/if}
	{/snippet}

	{#snippet content({ closeSheet })}
		{#if form}
			{@render form({ closeSheet })}
		{:else}
			<CollectionForm
				collection={manager.collection}
				{...options}
				onSuccess={(record) => {
					closeSheet();
					manager.loadRecords();
					onSuccess(record, 'create');
				}}
				uiOptions={{
					hide: ['submit_button', 'error']
				}}
				refineSchema={formRefineSchema}
			>
				<FormError />
				<div
					class="sticky bottom-0 -mx-6 -mt-6 flex justify-end border-t bg-white/70 px-6 py-2 backdrop-blur-sm"
				>
					<SubmitButton>
						{@render SubmitButtonText()}
					</SubmitButton>
				</div>
			</CollectionForm>
		{/if}
	{/snippet}
</Sheet>

{#snippet SubmitButtonText()}
	{#if buttonText}
		{@render buttonText?.()}
	{:else}
		{m.Create_record()}
	{/if}
{/snippet}
