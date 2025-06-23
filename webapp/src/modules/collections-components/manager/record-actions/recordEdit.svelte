<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="C extends CollectionName">
	import Pencil from 'lucide-svelte/icons/pencil';
	import type {
		CollectionFormData,
		CollectionResponses,
		CredentialsResponse
	} from '@/pocketbase/types';
	import { m } from '@/i18n';
	import type {
		CollectionFormOptions,
		FieldsOptions
	} from '@/collections-components/form/collectionFormTypes';
	import type { CollectionName } from '@/pocketbase/collections-models';
	import { getCollectionManagerContext } from '../collectionManagerContext';
	import { CollectionForm } from '@/collections-components';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import { merge } from 'lodash';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import type { RecordCreateEditProps } from './types';
	import { QrCode } from '@/qr';
	import type { SuperForm } from 'sveltekit-superforms';
	import { tick } from 'svelte';

	//

	type Props = RecordCreateEditProps<C> & {
		record: CollectionResponses[C];
		collection?: CollectionName;
		fieldsOptions?: Partial<FieldsOptions<C>>;
	};

	const {
		formTitle,
		onSuccess = () => {},
		buttonText,
		button,
		record,
		collection,
		fieldsOptions = {}
	}: Props = $props();

	//

	let form = $state<SuperForm<CollectionFormData[C]>>();
	let deeplink = $state('');

	const credentialRecord: CredentialsResponse | null = $derived(
		collection === 'credentials' ? (record as CredentialsResponse) : null
	);

	const { manager, formsOptions } = $derived(getCollectionManagerContext());

	const defaultFormOptions: CollectionFormOptions<C> = {
		uiOptions: { showToastOnSuccess: true }
	};
	const options = $derived(merge(defaultFormOptions, formsOptions.base, formsOptions.edit));

	const sheetTitle = $derived(formTitle ?? m.Edit_record());

	//

	$effect(() => {
		if (form && collection === 'credentials') {
			const unsubscribe = form.form.subscribe((d) => {
				if ('deeplink' in d) {
					deeplink = (d as { deeplink: string }).deeplink;
					moveQr();
				} else {
					deeplink = '';
				}
			});
			return unsubscribe;
		}
	});

	async function moveQr() {
		await tick(); // wait for Svelte to paint the opened sheet
		const qrImg = document.querySelector<HTMLImageElement>('img.size-40.rounded-sm');
		const deeplinkInput = document.querySelector<HTMLInputElement>('input[label="Deeplink"]');
		if (qrImg && deeplinkInput && deeplinkInput.parentNode) {
			deeplinkInput.parentNode.insertBefore(qrImg, deeplinkInput);
		}
	}

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
		{#if deeplink}
			<QrCode src={deeplink} class="size-40 rounded-sm" />
		{/if}
		<CollectionForm
			bind:form
			collection={collection || manager.collection}
			recordId={record.id}
			initialData={record}
			{...options}
			{...fieldsOptions ? { fieldsOptions } : {}}
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
