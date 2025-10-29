<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import type { CollectionField as PbCollectionField } from 'pocketbase';

	import type { AnyCollectionField, CollectionName } from '@/pocketbase/collections-models';

	import type { FieldSnippet, RelationFieldOptions } from './collectionFormTypes';

	export type CollectionFormFieldProps<C extends CollectionName> = {
		fieldConfig: PbCollectionField;
		hidden?: boolean;
		label?: string;
		snippet?: FieldSnippet<C>;
		relationFieldOptions?: RelationFieldOptions<C>;
		description?: string;
		placeholder?: string;
		recordId?: string;
		collectionName: string;
	};
</script>

<script lang="ts" generics="C extends CollectionName">
	import type { FormPath, SuperForm } from 'sveltekit-superforms';

	import type { CollectionFormData } from '@/pocketbase/types';

	import { getFormContext } from '@/forms';
	import { CheckboxField, Field, FileField, SelectField, TextareaField } from '@/forms/fields';
	import MarkdownField from '@/forms/fields/markdownField.svelte';
	import { getCollectionNameFromId, isArrayField } from '@/pocketbase/collections-models';

	import CollectionField from '../collectionField.svelte';

	//

	let {
		fieldConfig,
		label = fieldConfig.name,
		description,
		placeholder,
		hidden = false,
		snippet,
		relationFieldOptions = {},
		collectionName,
		recordId
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
		formData: $formData as CollectionFormData[C],
		recordId,
		collectionName
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
	<MarkdownField {form} {name} height={80} />
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
