<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Form } from '@/forms';
	import { TestConfigJsonForm } from './test-config-json-form.svelte.js';
	import { CodeEditorField, PlaceholderHighlightCodeEditorField } from '@/forms/fields/index.js';

	type Props = {
		form: TestConfigJsonForm;
	};

	const { form }: Props = $props();
</script>

<Form form={form.superform} hide={['submit_button']} hideRequiredIndicator>
	<div class="flex min-w-0 shrink-0 grow basis-1 flex-col">
		{#if form.isTainted}
			<CodeEditorField
				form={form.superform}
				name="json"
				options={{
					lang: 'json',
					label: 'JSON configuration',
					class: 'self-stretch'
				}}
			/>
		{:else}
			<PlaceholderHighlightCodeEditorField
				form={form.superform}
				name="json"
				fieldValues={form.placeholdersValues}
				options={{
					lang: 'json',
					label: 'JSON configuration',
					class: 'self-stretch'
				}}
			/>
		{/if}
	</div>
</Form>
