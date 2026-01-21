<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="C extends CollectionName">
	import { Plus } from '@lucide/svelte';
	import { merge } from 'lodash';

	import type { CollectionName } from '@/pocketbase/collections-models';

	import { CollectionForm } from '@/collections-components';
	import { type CollectionFormOptions } from '@/collections-components/form/collectionFormTypes';
	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import { FormError } from '@/forms';
	import { m } from '@/i18n';

	import type { RecordCreateEditProps } from './types';

	import { getCollectionManagerContext } from '../collectionManagerContext';
	import SubmitButton from './submit-button.svelte';

	//

	const {
		formTitle,
		onSuccess = () => {},
		buttonText,
		button,
		form
	}: RecordCreateEditProps<C> = $props();

	const { manager, formsOptions, formRefineSchema, createForm } = $derived(
		getCollectionManagerContext()
	);

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
		{:else if createForm}
			{@render createForm({ closeSheet })}
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

				<SubmitButton>
					{@render SubmitButtonText()}
				</SubmitButton>
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
