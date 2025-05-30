<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { TestConfigFieldsFormComponent } from '$lib/start-checks-form/test-config-fields-form';
	import { TestConfigFormComponent } from '$lib/start-checks-form/test-config-form';
	import Button from '@/components/ui-custom/button.svelte';
	import Footer from '../_utils/footer.svelte';
	import SectionCard from '../_utils/section-card.svelte';
	import { ChecksConfigForm, type ChecksConfigFormProps } from './checks-configs-form.svelte.js';
	import { m } from '@/i18n';

	//

	const props: ChecksConfigFormProps = $props();
	const form = new ChecksConfigForm(props);

	const SHARED_FIELDS_ID = 'shared-fields';
</script>

<div class="space-y-4">
	{#if form.hasSharedFields}
		<SectionCard id={SHARED_FIELDS_ID} title={m.Shared_fields()}>
			<TestConfigFieldsFormComponent form={form.sharedFieldsForm} />
		</SectionCard>
	{/if}

	{#each Object.entries(form.checksForms) as [id, checkForm]}
		<SectionCard {id} title={id.replace('.json', '')}>
			<TestConfigFormComponent form={checkForm} />
		</SectionCard>
	{/each}
</div>

<Footer>
	{#snippet left()}
		{@const status = form.getCompletionStatus()}

		<div>
			{#if form.hasSharedFields}
				<div class="flex items-center gap-2">
					{#if status.sharedFields}
						<p class="text-green-500">
							{m.Shared_fields()}
						</p>
					{:else}
						<p class="text-red-500">
							missing {status.missingSharedFieldsCount} shared fields
						</p>
					{/if}
				</div>
			{/if}

			<div>
				<p>Individual checks:</p>
				<p class="text-green-500">
					{status.validFormsCount}/{status.totalForms} valid forms
				</p>
				<p class="text-red-500">
					{status.invalidFormsCount}/{status.totalForms} invalid forms
				</p>
			</div>
		</div>
	{/snippet}

	{#snippet right()}
		<Button>ao</Button>
	{/snippet}
</Footer>
