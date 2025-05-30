<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Form } from '@/forms';
	import TestConfigFieldInput from './test-config-field/test-config-field-input.svelte';
	import type { TestConfigFieldsForm } from './test-config-fields-form.svelte.js';

	//

	type Props = {
		form: TestConfigFieldsForm;
	};

	let { form }: Props = $props();
</script>

<Form form={form.superform} hide={['submit_button']} hideRequiredIndicator>
	{#each form.props.fields as field}
		<TestConfigFieldInput {field} form={form.superform} />
	{/each}
</Form>
