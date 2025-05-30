<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { TestConfigFieldsFormComponent } from '$lib/start-checks-form/test-config-fields-form';
	import { TestConfigFormComponent } from '$lib/start-checks-form/test-config-form';
	import type { ChecksConfigForm } from './checks-configs-form.svelte.js';
	import { m } from '@/i18n';

	//

	type Props = {
		form: ChecksConfigForm;
	};

	const { form }: Props = $props();

	const SHARED_FIELDS_ID = 'shared-fields';
</script>

<div class="space-y-4">
	<h2 id={SHARED_FIELDS_ID} class="text-lg font-bold">{m.Shared_fields()}</h2>
	<TestConfigFieldsFormComponent form={form.sharedFieldsForm} />
</div>

{#each Object.entries(form.checksForms) as [id, checkForm]}
	<div class="space-y-4">
		<h2 {id} class="text-lg font-bold">{id}</h2>
		<TestConfigFormComponent form={checkForm} />
	</div>
{/each}
