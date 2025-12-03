<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="Data extends GenericRecord, T extends SelectType">
	import type { Writable } from 'svelte/store';
	import type { SuperForm } from 'sveltekit-superforms';

	import { fieldProxy, type FormPath } from 'sveltekit-superforms/client';

	import type { MaybeArray } from '@/utils/other';
	import type { GenericRecord } from '@/utils/types';

	import SelectInput, {
		type SelectProps,
		type SelectType
	} from '@/components/ui-custom/selectInput.svelte';
	import * as Form from '@/components/ui/form';

	import type { FieldOptions } from './types';

	import FieldWrapper from './parts/fieldWrapper.svelte';

	//

	interface Props {
		form: SuperForm<Data>;
		name: FormPath<Data>;
		options: Partial<FieldOptions> & Partial<SelectProps<T>>;
	}

	const { form, name, options = {} }: Props = $props();
	const { type = 'single' as SelectType, items = [], trigger, ...rest } = $derived(options);

	//

	const value = fieldProxy(form, name) as Writable<MaybeArray<string>>;

	// TODO - Fix types
	// - trigger={trigger as undefined}
	// - value={$value as unknown as undefined}
</script>

<Form.Field {form} {name}>
	<FieldWrapper field={name} options={{ label: options.label, description: options.description }}>
		{#snippet children({ props })}
			<SelectInput
				{...rest}
				{items}
				{type}
				value={$value as unknown as undefined}
				controlAttrs={props}
				trigger={trigger as undefined}
				onValueChange={(data) => ($value = data)}
			/>
		{/snippet}
	</FieldWrapper>
</Form.Field>
