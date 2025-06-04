<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Form } from '@/forms';
	import ConfigFormInput from './config-form-input.svelte';
	import type { TestConfigFieldsForm } from './config-form-fields.js';

	//

	type Props = {
		form: TestConfigFieldsForm;
	};

	let { form }: Props = $props();
</script>

<Form form={form.superform} hide={['submit_button']} class="space-y-6" hideRequiredIndicator>
	{#each form.props.fields as field}
		<ConfigFormInput {field} form={form.superform} />
	{/each}
</Form>
