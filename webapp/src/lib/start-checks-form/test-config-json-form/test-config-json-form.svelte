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
	<PlaceholderHighlightCodeEditorField
		form={form.superform}
		name="json"
		fieldValues={form.placeholdersValues}
		options={{
			lang: 'json',
			class: 'self-stretch',
			hideLabel: true
		}}
	/>
</Form>
