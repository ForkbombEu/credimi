<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="Data extends GenericRecord">
	import type { ComponentProps } from 'svelte';
	import type { FormPathLeaves, SuperForm } from 'sveltekit-superforms';

	import { formFieldProxy } from 'sveltekit-superforms';

	import type { GenericRecord } from '@/utils/types';

	import CodeEditor from '@/components/ui-custom/codeEditor.svelte';
	import * as Form from '@/components/ui/form';

	import type { FieldOptions } from './types';

	import FieldWrapper from './parts/fieldWrapper.svelte';

	//

	interface Props {
		form: SuperForm<Data>;
		name: FormPathLeaves<Data, string | number>;
		options: Partial<FieldOptions> & ComponentProps<typeof CodeEditor>;
	}

	const { form, name, options }: Props = $props();

	const { validate } = form;
	const { value: v } = formFieldProxy(form, name);
</script>

<Form.Field {form} {name}>
	<FieldWrapper field={name} {options}>
		<CodeEditor
			{...options}
			bind:value={$v as string}
			onBlur={() => {
				validate(name);
			}}
		/>
	</FieldWrapper>
</Form.Field>
