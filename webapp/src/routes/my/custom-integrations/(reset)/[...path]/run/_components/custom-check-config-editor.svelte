<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { yaml } from '@codemirror/lang-yaml';

	import { JsonSchemaFormComponent } from '@/components/json-schema-form';
	import Button from '@/components/ui-custom/button.svelte';
	import { CodeEditorField } from '@/forms/fields/index.js';
	import { Form } from '@/forms/index.js';
	import { m } from '@/i18n';

	import type { CustomCheckConfigEditor } from './custom-check-config-editor.svelte.js';

	const { editor }: { editor: CustomCheckConfigEditor } = $props();
</script>

<div class="space-y-6">
	<div class="flex gap-6">
		{#if editor.jsonSchemaForm}
			<div class="grow basis-1 space-y-6">
				<h3 class="text-sm font-medium">{m.Fields()}</h3>
				<JsonSchemaFormComponent form={editor.jsonSchemaForm} hideSubmitButton />
			</div>
		{/if}
		<div class="min-w-0 grow basis-1 space-y-6">
			<h3 class="text-sm font-medium">{m.YAML_Configuration()}</h3>
			<Form form={editor.yamlForm} hide={['submit_button']}>
				<CodeEditorField form={editor.yamlForm} name="yaml" options={{ lang: yaml() }} />
			</Form>
		</div>
	</div>

	<div class="flex justify-end">
		<Button disabled={!editor.isValid} onclick={() => editor.submit()}>
			{m.Run_integration()}
		</Button>
	</div>
</div>
