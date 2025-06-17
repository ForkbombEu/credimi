<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { JsonSchemaFormComponent } from '@/components/json-schema-form';
	import type { CustomCheckConfigEditor } from './custom-check-config-editor.svelte.js';
	import { yaml } from '@codemirror/lang-yaml';
	import SmallSectionLabel from '$start-checks-form/_utils/small-section-label.svelte';
	import { m } from '@/i18n';
	import { Form } from '@/forms/index.js';
	import { CodeEditorField } from '@/forms/fields/index.js';

	const { editor }: { editor: CustomCheckConfigEditor } = $props();
</script>

<div class="flex gap-6">
	<div class="grow basis-1 space-y-6">
		<SmallSectionLabel>{m.Fields()}</SmallSectionLabel>
		<JsonSchemaFormComponent form={editor.jsonSchemaForm} hideSubmitButton />
	</div>
	<div class="min-w-0 grow basis-1 space-y-6">
		<SmallSectionLabel>{m.YAML_Configuration()}</SmallSectionLabel>
		<Form form={editor.yamlForm} hide={['submit_button']}>
			<CodeEditorField form={editor.yamlForm} name="yaml" options={{ lang: yaml() }} />
		</Form>
	</div>
</div>
