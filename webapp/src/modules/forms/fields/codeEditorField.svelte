<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="Data extends GenericRecord">
	import type { ComponentProps } from 'svelte';
	import type { FormPathLeaves, SuperForm } from 'sveltekit-superforms';

	import { formFieldProxy } from 'sveltekit-superforms';

	// eslint-disable-next-line @typescript-eslint/no-unused-vars
	import type { GenericRecord } from '@/utils/types';

	import CodeEditor from '@/components/ui-custom/codeEditor.svelte';
	import CodeEditorWithOutput from '@/components/ui-custom/codeEditorWithOutput.svelte';
	import * as Form from '@/components/ui/form';

	import type { FieldOptions } from './types';

	import FieldWrapper from './parts/fieldWrapper.svelte';

	//

	interface Props {
		form: SuperForm<Data>;
		name: FormPathLeaves<Data, string | number>;
		options: Partial<FieldOptions> &
			ComponentProps<typeof CodeEditor> & {
				useOutput?: boolean;
				onRun?: (code: string) => Promise<unknown> | unknown;
				output?: string;
				error?: string;
				running?: boolean;
			};
	}

	const { form, name, options }: Props = $props();

	const { validate } = form;
	const { value } = formFieldProxy(form, name);
</script>

<Form.Field {form} {name}>
	<FieldWrapper field={name} {options}>
		{#if options.useOutput}
			<CodeEditorWithOutput
				bind:value={$value as string}
				onRun={options.onRun}
				output={options.output}
				error={options.error}
				running={options.running}
			/>
		{:else}
			<CodeEditor
				{...options}
				bind:value={$value as string}
				onBlur={() => {
					validate(name);
				}}
			/>
		{/if}
	</FieldWrapper>
</Form.Field>
