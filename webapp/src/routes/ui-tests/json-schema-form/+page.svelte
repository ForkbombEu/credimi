<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import JsonSchemaForm from '@/components/json-schema-form.svelte';
</script>

<JsonSchemaForm
	schema={{
		type: 'object',
		title: 'Form title',
		properties: {
			text: {
				type: 'string',
				title: 'Text input'
			},
			number: {
				type: 'number',
				title: 'Number input'
			},
			boolean: {
				type: 'boolean',
				title: 'Boolean input'
			}
		},
		required: ['text']
	}}
/>
