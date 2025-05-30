<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { TestConfigFieldsFormComponent } from '$lib/start-checks-form/test-config-fields-form';
	import { TestConfigFormComponent } from '$lib/start-checks-form/test-config-form';
	import { ChecksConfigForm, type ChecksConfigFormProps } from './checks-configs-form.svelte.js';
	import { m } from '@/i18n';

	//

	const props: ChecksConfigFormProps = $props();
	const form = new ChecksConfigForm(props);

	const SHARED_FIELDS_ID = 'shared-fields';
</script>

<div id={SHARED_FIELDS_ID} class="space-y-4">
	<h2 class="text-lg font-bold">{m.Shared_fields()}</h2>
	<TestConfigFieldsFormComponent form={form.sharedFieldsForm} />
</div>

{#each Object.entries(form.checksForms) as [id, checkForm]}
	<div {id} class="space-y-4">
		<h2 class="text-lg font-bold">{id}</h2>
		<TestConfigFormComponent form={checkForm} />
	</div>
{/each}
