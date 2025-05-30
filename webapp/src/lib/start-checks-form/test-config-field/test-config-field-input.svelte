<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CodeEditorField, Field } from '@/forms/fields';
	import type { SuperForm } from 'sveltekit-superforms';
	import type { TestConfigField } from './test-config-field';
	import type { Snippet } from 'svelte';
	import type { StringRecord } from '@/utils/types';

	//

	type Props = {
		field: TestConfigField;
		form: SuperForm<StringRecord>;
		labelRight?: Snippet;
	};

	const { field, form, labelRight }: Props = $props();
</script>

{#if field.Type == 'string'}
	<Field {form} name={field.CredimiID} options={{ label: field.LabelKey, labelRight }} />
{:else if field.Type == 'object'}
	<CodeEditorField
		{form}
		name={field.CredimiID}
		options={{
			lang: 'json',
			label: field.LabelKey,
			value: field.Example,
			labelRight
		}}
	/>
{/if}
