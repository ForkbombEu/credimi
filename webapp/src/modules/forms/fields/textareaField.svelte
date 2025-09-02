<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="Data extends GenericRecord">
	import type { ComponentProps } from 'svelte';
	import type { FormPathLeaves, SuperForm } from 'sveltekit-superforms';

	import { stringProxy } from 'sveltekit-superforms';

	import type { GenericRecord } from '@/utils/types';

	import * as Form from '@/components/ui/form';
	import { Textarea } from '@/components/ui/textarea';

	import type { FieldOptions } from './types';

	import FieldWrapper from './parts/fieldWrapper.svelte';

	//

	interface Props {
		form: SuperForm<Data>;
		name: FormPathLeaves<Data, string | number>;
		options?: Partial<FieldOptions> & ComponentProps<typeof Textarea>;
	}

	const { form, name, options = {} }: Props = $props();

	const { form: formData } = $derived(form);
	const valueProxy = $derived(stringProxy(formData, name, { empty: 'undefined' }));
</script>

<Form.Field {form} {name}>
	<FieldWrapper field={name} {options}>
		{#snippet children({ props })}
			<Textarea {...options} {...props} bind:value={$valueProxy} />
		{/snippet}
	</FieldWrapper>
</Form.Field>
