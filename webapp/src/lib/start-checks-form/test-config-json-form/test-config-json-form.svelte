<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Form } from '@/forms';
	import { TestConfigJsonForm } from './test-config-json-form.svelte.js';
	import { CodeEditorField } from '@/forms/fields/index.js';
	import {
		refreshEditorView,
		type PlaceholderData,
		displayPlaceholderData
	} from './highlight-plugin.js';

	import { isNamedTestConfigField } from '$lib/start-checks-form/test-config-field';
	import type { EditorView } from '@codemirror/view';

	//

	type Props = {
		form: TestConfigJsonForm;
	};

	const { form }: Props = $props();

	// Placeholders update preview

	let editorView = $state<EditorView>();

	function getPlaceholdersData(): PlaceholderData[] {
		const { formDependency } = form.props;
		if (!formDependency) return [];

		const { validData } = formDependency.getCompletionReport();
		return formDependency.props.fields.filter(isNamedTestConfigField).map((field) => {
			return {
				field,
				isValid: field.CredimiID in validData,
				value: validData[field.CredimiID]
			};
		});
	}

	$effect(() => {
		if (!editorView) return;
		getPlaceholdersData();
		refreshEditorView(editorView);
	});
</script>

<Form form={form.superform} hide={['submit_button']} hideRequiredIndicator>
	<CodeEditorField
		form={form.superform}
		name="json"
		options={{
			lang: 'json',
			class: 'self-stretch',
			hideLabel: true,
			extensions: [
				displayPlaceholderData({
					placeholdersRegex: /"?\{\{\s*\.(\w+)\s*\}\}"?/g,
					getPlaceholdersData
				})
			],
			onReady: (view) => {
				editorView = view;
			}
		}}
	/>
</Form>
