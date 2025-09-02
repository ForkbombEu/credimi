<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ComponentProps } from 'svelte';

	import SuperDebug from 'sveltekit-superforms';
	import { zod } from 'sveltekit-superforms/adapters';

	import { CollectionField } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import { createForm, Form } from '@/forms';
	import {
		Field,
		SwitchField,
		FileField,
		CheckboxField,
		TextareaField,
		SelectField
	} from '@/forms/fields';
	import { ZTestCollectionSelectFieldOptions } from '@/pocketbase/types';
	import { createCollectionZodSchema } from '@/pocketbase/zod-schema';
	import { createDummyFile } from '@/utils/other';

	const schema = createCollectionZodSchema('z_test_collection');
	console.log(schema.shape);

	const form = createForm({
		adapter: zod(schema),
		onSubmit: async () => {
			await new Promise((res) => setTimeout(() => res('p1'), 2000));
			throw new Error('no wayy');
		},
		initialData: {
			file_field: createDummyFile(),
			relation_field: '5bxdhl7ieg0ogbd'
		},
		options: {
			dataType: 'form'
		}
	});

	const { form: formData } = form;

	const selectItems = Object.values(ZTestCollectionSelectFieldOptions).map((v) => ({
		label: v,
		value: v as string
	}));

	//

	let hide = $state<boolean>(false);

	// function changeHide() {
	// 	hide = !hide;
	// }
</script>

<!-- <Button onclick={changeHide}>sad</Button> -->

<Form {form} hideRequiredIndicator={hide}>
	<Field {form} name="text_field" />

	<SelectField {form} name="select_field" options={{ items: selectItems }} />
	<SelectField
		{form}
		name="select_multi_field"
		options={{ items: selectItems, type: 'multiple' }}
	/>

	<CollectionField
		{form}
		name="relation_field"
		collection="z_test_collection"
		options={{
			mode: 'search',
			displayFields: ['text_field']
		}}
	/>

	<CollectionField
		{form}
		name="relation_multi_field"
		collection="z_test_collection"
		options={{
			mode: 'search',
			multiple: true,
			displayFields: ['number_field']
		}}
	/>

	<CollectionField {form} name="relation_field" collection="z_test_collection" />

	<CollectionField
		{form}
		name="relation_multi_field"
		collection="z_test_collection"
		options={{
			multiple: true,
			displayFields: ['text_field']
		}}
	/>

	<SuperDebug data={formData}></SuperDebug>

	<Field {form} name="url_field" options={{ type: 'url' }} />
	<Field {form} name="number_field" options={{ type: 'number' }} />
	<FileField {form} name="file_field" />
	<FileField {form} name="file_multi_field" options={{ multiple: true }} />
	<Field {form} name="date_field" options={{ type: 'date' }} />
	<SwitchField {form} name="boolean_field" />
	<CheckboxField {form} name="boolean_field" />
	<TextareaField {form} name="json_field" />
</Form>
