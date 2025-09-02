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
	import { m } from '@/i18n';

	import type { RecordCreateEditProps } from './types';

	import { getCollectionManagerContext } from '../collectionManagerContext';

	//

	const {
		formTitle,
		onSuccess = () => {},
		buttonText,
		button
	}: RecordCreateEditProps<C> = $props();

	const { manager, formsOptions } = $derived(getCollectionManagerContext());

	const defaultFormOptions: CollectionFormOptions<C> = {
		uiOptions: { showToastOnSuccess: true }
	};
	const options = $derived(merge(defaultFormOptions, formsOptions.base, formsOptions.edit));

	const sheetTitle = $derived(formTitle ?? m.Create_record());
</script>

<Sheet title={sheetTitle}>
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
		<CollectionForm
			collection={manager.collection}
			{...options}
			onSuccess={(record) => {
				closeSheet();
				manager.loadRecords();
				onSuccess(record, 'create');
			}}
		>
			{#snippet submitButtonContent()}
				{@render SubmitButtonText()}
			{/snippet}
		</CollectionForm>
	{/snippet}
</Sheet>

{#snippet SubmitButtonText()}
	{#if buttonText}
		{@render buttonText?.()}
	{:else}
		{m.Create_record()}
	{/if}
{/snippet}
