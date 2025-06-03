<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { JsonSchemaFormComponent } from '@/components/json-schema-form';
	import type { CustomCheckForm } from './custom-check-form.svelte.js';
	import CodeEditor from '@/components/ui-custom/codeEditor.svelte';
	import { yaml } from '@codemirror/lang-yaml';
	import SmallSectionLabel from '$lib/start-checks-form/_utils/small-section-label.svelte';
	import { m } from '@/i18n';

	const { form }: { form: CustomCheckForm } = $props();
</script>

<div class="flex gap-6">
	<div class="grow basis-1 space-y-6">
		<SmallSectionLabel>{m.Fields()}</SmallSectionLabel>
		<JsonSchemaFormComponent form={form.jsonSchemaForm} hideSubmitButton />
	</div>
	<div class="min-w-0 grow basis-1 space-y-6">
		<SmallSectionLabel>{m.YAML_Configuration()}</SmallSectionLabel>
		<CodeEditor lang={yaml()} value={form.props.customCheck.yaml} />
	</div>
</div>
