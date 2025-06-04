<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Form } from '@/forms';
	import { CheckConfigJsonEditor } from './check-config-json-editor.svelte.js';
	import { CodeEditorField } from '@/forms/fields/index.js';
	import {
		dispatchEffect,
		type PlaceholderData,
		displayPlaceholderData
	} from './highlight-plugin.js';

	import { isNamedConfigField } from '$start-checks-form/_utils';
	import type { EditorView } from '@codemirror/view';

	//

	type Props = {
		editor: CheckConfigJsonEditor;
	};

	const { editor }: Props = $props();

	// Placeholders update preview

	let codeEditorView = $state<EditorView>();

	function getPlaceholdersData(): PlaceholderData[] {
		const { editorDependency: formDependency } = editor.props;
		if (!formDependency) return [];

		const { validData } = formDependency.getCompletionReport();
		return formDependency.props.fields.filter(isNamedConfigField).map((field) => {
			return {
				field,
				isValid: field.CredimiID in validData,
				value: validData[field.CredimiID]
			};
		});
	}

	$effect(() => {
		if (!codeEditorView) return;
		getPlaceholdersData();
		dispatchEffect(codeEditorView, 'updatePlaceholders');
	});

	$effect(() => {
		if (!codeEditorView) return;
		if (!editor.isTainted) return;
		dispatchEffect(codeEditorView, 'removePlaceholders');
	});
</script>

<Form form={editor.superform} hide={['submit_button']} hideRequiredIndicator>
	<CodeEditorField
		form={editor.superform}
		name="json"
		options={{
			lang: 'json',
			class: 'self-stretch',
			hideLabel: true,
			maxHeight: 600,
			extensions: [
				displayPlaceholderData({
					placeholdersRegex: /"?\{\{\s*\.(\w+)\s*\}\}"?/g,
					getPlaceholdersData
				})
			],
			onReady: (view) => {
				codeEditorView = view;
			}
		}}
	/>
</Form>
