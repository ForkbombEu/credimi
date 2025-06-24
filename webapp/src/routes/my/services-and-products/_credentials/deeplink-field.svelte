<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="Data extends GenericRecord">
	import type { GenericRecord } from '@/utils/types';
	import * as Form from '@/components/ui/form';
	import type { FormPathLeaves, SuperForm } from 'sveltekit-superforms';
	import { stringProxy } from 'sveltekit-superforms';
	import FieldWrapper from '@/forms/fields/parts/fieldWrapper.svelte';
	import type { FieldOptions } from '@/forms/fields/types';
	import MarkdownEditor from '@/components/ui-custom/markdown-editor.svelte';

	//

	interface Props {
		form: SuperForm<Data>;
		name: FormPathLeaves<Data, string>;
		options?: Partial<FieldOptions>;
	}

	const { form, name, options = {} }: Props = $props();

	const { form: formData } = $derived(form);
	const valueProxy = $derived(stringProxy(formData, name, { empty: 'undefined' }));
</script>

<Form.Field {form} {name}>
	<FieldWrapper field={name} {options}>
		{#snippet children()}
			<MarkdownEditor bind:value={$valueProxy} />
		{/snippet}
	</FieldWrapper>
</Form.Field>
