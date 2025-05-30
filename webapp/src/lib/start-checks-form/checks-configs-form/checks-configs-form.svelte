<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { TestConfigFieldsFormComponent } from '$lib/start-checks-form/test-config-fields-form';
	import { TestConfigFormComponent } from '$lib/start-checks-form/test-config-form';
	import SectionCard from '../_utils/section-card.svelte';
	import { ChecksConfigForm, type ChecksConfigFormProps } from './checks-configs-form.svelte.js';
	import { m } from '@/i18n';

	//

	const props: ChecksConfigFormProps = $props();
	const form = new ChecksConfigForm(props);

	const SHARED_FIELDS_ID = 'shared-fields';
</script>

<div class="space-y-4">
	{#if form.props.normalized_fields.length > 0}
		<SectionCard id={SHARED_FIELDS_ID} title={m.Shared_fields()}>
			<TestConfigFieldsFormComponent form={form.sharedFieldsForm} />
		</SectionCard>
	{/if}

	{#each Object.entries(form.checksForms) as [id, checkForm]}
		<SectionCard {id} title={id}>
			<TestConfigFormComponent form={checkForm} />
		</SectionCard>
	{/each}
</div>
