<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CodeEditorField, Field } from '@/forms/fields';
	import type { SuperForm } from 'sveltekit-superforms';
	import type { FieldConfig } from './tests-configs-form/types';

	//

	type Props = {
		config: FieldConfig;
		form: SuperForm<Record<string, unknown>>;
	};

	const { config, form }: Props = $props();
</script>

{#if config.Type == 'string'}
	<Field {form} name={config.CredimiID} options={{ label: config.LabelKey }} />
{:else if config.Type == 'object'}
	<CodeEditorField
		{form}
		name={config.CredimiID}
		options={{
			lang: 'json',
			label: config.LabelKey,
			value: config.Example
		}}
	/>
{/if}
