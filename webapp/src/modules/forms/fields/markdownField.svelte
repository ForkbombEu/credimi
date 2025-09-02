<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="Data extends GenericRecord">
	import type { FormPathLeaves, SuperForm } from 'sveltekit-superforms';

	import { stringProxy } from 'sveltekit-superforms';

	import type { GenericRecord } from '@/utils/types';

	import MarkdownEditor from '@/components/ui-custom/markdown-editor.svelte';
	import * as Form from '@/components/ui/form';

	import type { FieldOptions } from './types';

	import FieldWrapper from './parts/fieldWrapper.svelte';

	//

	interface Props {
		form: SuperForm<Data>;
		name: FormPathLeaves<Data, string>;
		options?: Partial<FieldOptions>;
		height?: number;
	}

	const { form, name, options = {}, height }: Props = $props();

	const { form: formData } = $derived(form);
	const valueProxy = $derived(stringProxy(formData, name, { empty: 'undefined' }));
</script>

<Form.Field {form} {name}>
	<FieldWrapper field={name} {options}>
		{#snippet children()}
			<MarkdownEditor bind:value={$valueProxy} {height} />
		{/snippet}
	</FieldWrapper>
</Form.Field>
