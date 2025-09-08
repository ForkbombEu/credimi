<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { CollectionField } from 'pocketbase';

	import { form as f, pocketbase as pb } from '#';

	import type { CollectionFormData } from '@/pocketbase/types';
	import type { KeyOf } from '@/utils/types';

	import CollectionFormField, {
		type CollectionFormFieldProps
	} from '@/collections-components/form/collectionFormField.svelte';
	import {
		type CollectionFormProps,
		type FieldsOptions
	} from '@/collections-components/form/collectionFormTypes';
	import { getCollectionFields } from '@/pocketbase/zod-schema';
	import { capitalize } from '@/utils/other';

	//

	type Props = CollectionFormProps<C> & {
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		form: pb.recordform.Instance<any>;
	};

	// eslint-disable-next-line svelte/valid-compile
	let { form, ...props }: Props = $props();

	const { collection, fieldsOptions = {} } = $derived(props);

	// eslint-disable-next-line no-undef
	type F = FieldsOptions<C>;

	const {
		order: fieldsOrder = [],
		exclude: excludeFields = [] as string[],
		hide = {} as F['hide'],
		labels = {} as F['labels'],
		descriptions = {} as F['descriptions'],
		placeholders = {} as F['placeholders'],
		snippets = {} as F['snippets'],
		relations = {} as F['relations']
	} = $derived(fieldsOptions);

	/* Fields */

	const fieldsConfigs = $derived(
		getCollectionFields(collection)
			.sort(createFieldConfigSorter(fieldsOrder))
			.filter((config) => !excludeFields.includes(config.name))
	);

	function createFieldConfigSorter(order: string[] = []) {
		return (a: CollectionField, b: CollectionField) => {
			const aIndex = order.indexOf(a.name);
			const bIndex = order.indexOf(b.name);
			if (aIndex === -1 && bIndex === -1) {
				return 0;
			}
			if (aIndex === -1) {
				return 1;
			}
			if (bIndex === -1) {
				return -1;
			}
			return aIndex - bIndex;
		};
	}

	const fields = $derived<CollectionFormFieldProps<C>[]>(
		fieldsConfigs.map((fieldConfig) => {
			const name = fieldConfig.name as KeyOf<CollectionFormData[C]>;

			return {
				fieldConfig,
				hidden: Object.keys(hide).includes(name),
				label: labels[name] ?? capitalize(name),
				snippet: snippets[name],
				relationFieldOptions: relations[name],
				description: descriptions[name],
				placeholder: placeholders[name]
			};
		})
	);
</script>

{#key form.currentMode}
	{#if form.form}
		<f.Component form={form.form}>
			{#each fields as field}
				<CollectionFormField {...field} />
			{/each}
		</f.Component>
	{/if}
{/key}
