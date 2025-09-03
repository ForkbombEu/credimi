<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="Data extends GenericRecord">
	import type { HTMLInputAttributes } from 'svelte/elements';
	import type { FormPathLeaves, FormPathType, SuperForm } from 'sveltekit-superforms';

	import { numberProxy, fieldProxy } from 'sveltekit-superforms';

	import type { GenericRecord } from '@/utils/types';

	import * as Form from '@/components/ui/form';
	import { Input } from '@/components/ui/input';

	import type { FieldOptions } from './types';

	import FieldWrapper from './parts/fieldWrapper.svelte';

	//

	interface Props {
		form: SuperForm<Data>;
		name: FormPathLeaves<Data, string | number>;
		options?: Partial<FieldOptions & HTMLInputAttributes>;
	}

	const { form, name, options = {} }: Props = $props();

	//

	const { form: formData } = form;

	const numberValue = $derived(numberProxy(formData, name));
	const textValue = $derived(fieldProxy(formData, name));
	const valueProxy = $derived(options.type == 'number' ? numberValue : textValue);

	//

	const defaultPlaceholders: Record<string, string> = {
		text: 'abc',
		email: 'email@example.org',
		url: 'https://www.website.org'
	};

	let placeholder = $derived(
		options.placeholder ? options.placeholder : defaultPlaceholders[options.type ?? 'text']
	);
</script>

<Form.Field {form} {name}>
	<FieldWrapper field={name} {options}>
		{#snippet children({ props })}
			{#if valueProxy}
				<Input
					{...options as GenericRecord}
					{...props as GenericRecord}
					{placeholder}
					bind:value={$valueProxy}
				/>
			{/if}
		{/snippet}
	</FieldWrapper>
</Form.Field>
