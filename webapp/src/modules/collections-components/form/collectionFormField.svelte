<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import type { CollectionName, AnyCollectionField } from '@/pocketbase/collections-models';
	import type { FieldSnippet, RelationFieldOptions } from './collectionFormTypes';
	import type { CollectionField as PbCollectionField } from 'pocketbase';

	export type CollectionFormFieldProps<C extends CollectionName> = {
		fieldConfig: PbCollectionField;
		hidden?: boolean;
		label?: string;
		snippet?: FieldSnippet<C>;
		relationFieldOptions?: RelationFieldOptions<C>;
		description?: string;
		placeholder?: string;
	};
</script>

<script lang="ts" generics="C extends CollectionName">
	import { getFormContext } from '@/forms';
	import { CheckboxField, FileField, Field, SelectField, TextareaField } from '@/forms/fields';
	import CollectionField from '../collectionField.svelte';
	import { getCollectionNameFromId } from '@/pocketbase/collections-models';
	import { isArrayField } from '@/pocketbase/collections-models';
	import type { FormPath, SuperForm } from 'sveltekit-superforms';
	import type { CollectionFormData } from '@/pocketbase/types';

	//

	let {
		fieldConfig,
		label = fieldConfig.name,
		description,
		placeholder,
		hidden = false,
		snippet,
		relationFieldOptions = {}
	}: CollectionFormFieldProps<C> = $props();

	//

	const config = $derived(fieldConfig as AnyCollectionField);
	const name = $derived(config.name);
	const multiple = $derived(isArrayField(config));

	const { form } = getFormContext();
	const { form: formData } = form;
</script>

{#if hidden}
	<!-- Nothing -->
{:else if snippet}
	{@render snippet({
		form: form as unknown as SuperForm<CollectionFormData[C]>,
		field: name as FormPath<CollectionFormData[C]>,
		formData: $formData as CollectionFormData[C]
	})}
{:else if config.type == 'text' || config.type == 'url' || config.type == 'date' || config.type == 'email'}
	<Field {form} {name} options={{ label, description, placeholder, type: config.type }} />
{:else if config.type == 'number'}
	<Field {form} {name} options={{ label, description, type: 'number', placeholder }} />
{:else if config.type == 'json'}
	<TextareaField {form} {name} options={{ label, description, placeholder }} />
{:else if config.type == 'bool'}
	<CheckboxField {form} {name} options={{ label, description }} />
{:else if config.type == 'file'}
	{@const accept = config.mimeTypes?.join(',')}
	<FileField {form} {name} options={{ label, multiple, accept, placeholder }} />
{:else if config.type == 'select'}
	{@const items = config.values?.map((v) => ({ label: v, value: v }))}
	<SelectField
		{form}
		{name}
		options={{ label, items, type: multiple ? 'multiple' : 'single', description, placeholder }}
	/>
{:else if config.type == 'editor'}
	<TextareaField {form} {name} options={{ label, description, placeholder }} />
{:else if config.type == 'relation'}
	{@const collectionName = getCollectionNameFromId(config.collectionId) as C}
	<CollectionField
		{form}
		{name}
		collection={collectionName}
		options={{
			...relationFieldOptions,
			multiple,
			label,
			description
		}}
	/>
{/if}
