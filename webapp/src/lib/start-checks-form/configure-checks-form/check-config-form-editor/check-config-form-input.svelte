<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ConfigField } from '$start-checks-form/types';
	import type { Snippet } from 'svelte';
	import type { SuperForm } from 'sveltekit-superforms';

	import type { StringRecord } from '@/utils/types';

	import { CodeEditorField, Field } from '@/forms/fields';

	//

	type Props = {
		field: ConfigField;
		form: SuperForm<StringRecord>;
		labelRight?: Snippet;
	};

	const { field, form, labelRight }: Props = $props();
</script>

{#if field.field_type == 'string'}
	<Field {form} name={field.credimi_id} options={{ label: field.field_label, labelRight }} />
{:else if field.field_type == 'object'}
	<CodeEditorField
		{form}
		name={field.credimi_id}
		options={{
			lang: 'json',
			label: field.field_label,
			value: field.field_default_value,
			labelRight
		}}
	/>
{/if}
