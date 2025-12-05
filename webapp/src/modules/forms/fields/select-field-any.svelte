<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="Data extends GenericRecord, T">
	import type { SuperForm } from 'sveltekit-superforms';

	import { fieldProxy, type FormPath } from 'sveltekit-superforms/client';

	import type { SelectOption } from '@/components/ui-custom/utils';
	import type { GenericRecord } from '@/utils/types';

	import SelectInputAny from '@/components/ui-custom/select-input-any.svelte';
	import * as Form from '@/components/ui/form';

	import type { FieldOptions } from './types';

	import FieldWrapper from './parts/fieldWrapper.svelte';

	//

	interface Props {
		form: SuperForm<Data>;
		name: FormPath<Data>;
		options: Partial<FieldOptions> & {
			items: SelectOption<T>[];
			placeholder?: string;
		};
	}

	const { form, name, options }: Props = $props();

	//

	const value = fieldProxy(form, name);
</script>

<Form.Field {form} {name}>
	<FieldWrapper field={name} options={{ label: options.label, description: options.description }}>
		{#snippet children({ props })}
			<SelectInputAny
				items={options.items}
				bind:value={$value as undefined}
				placeholder={options.placeholder}
				controlAttrs={props}
			/>
		{/snippet}
	</FieldWrapper>
</Form.Field>
