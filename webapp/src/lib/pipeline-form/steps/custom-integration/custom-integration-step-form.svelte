<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SelfProp } from '$lib/renderable';

	import { userOrganization } from '$lib/app-state';
	import { entities } from '$lib/global/entities';

	import type { PocketbaseQueryResponse } from '@/pocketbase/query';
	import type { CustomChecksResponse } from '@/pocketbase/types';

	import { JsonSchemaFormComponent } from '@/components/json-schema-form';
	import T from '@/components/ui-custom/t.svelte';
	import { Button } from '@/components/ui/button';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import type { CustomIntegrationStepForm } from './custom-integration-step-form.svelte.js';

	import ItemCard from '../_partials/item-card.svelte';
	import StepCollectionPicker from '../_partials/step-collection-picker.svelte';
	import WithLabel from '../_partials/with-label.svelte';

	let { self: form }: SelfProp<CustomIntegrationStepForm> = $props();

	type CustomCheckWithOwner = PocketbaseQueryResponse<'custom_checks', ['owner']>;

	function ownerSubtitle(integration: CustomChecksResponse) {
		return (integration as CustomCheckWithOwner).expand?.owner?.name;
	}

	const orgId = $derived(userOrganization.current?.id ?? '');
	const pickerFilter = $derived(
		orgId ? `(owner.id = "${orgId}") || (published = true)` : 'published = true'
	);
</script>

{#if form.data.integration}
	<div class="space-y-6 border-b p-4">
		<WithLabel label={entities.custom_checks.labels.singular}>
			<ItemCard
				avatar={form.data.integration.logo
					? pb.files.getURL(form.data.integration, form.data.integration.logo)
					: undefined}
				title={form.data.integration.name}
				subtitle={ownerSubtitle(form.data.integration)}
				onDiscard={() => form.discardIntegration()}
			/>
		</WithLabel>

		{#if form.jsonSchemaForm}
			<div class="space-y-2">
				<div class="flex items-center gap-2">
					<hr class="grow border border-muted" />
					<h3 class="text-sm text-muted-foreground">{m.Configure_integration()}</h3>
					<hr class="grow border border-muted" />
				</div>
				<JsonSchemaFormComponent form={form.jsonSchemaForm} hideSubmitButton />
			</div>
		{/if}

		{#if form.intent === 'add' && form.hasSchema}
			<Button class="w-full" disabled={!form.isValid} onclick={() => form.submit()}>
				<T>{m.Add_step()}</T>
			</Button>
		{/if}
	</div>
{/if}

{#if form.state === 'select-integration'}
	<StepCollectionPicker
		collection="custom_checks"
		label={entities.custom_checks.labels.singular}
		queryOptions={{
			filter: pickerFilter,
			searchFields: ['name'],
			expand: ['owner']
		}}
		onSelect={(record) => form.selectIntegration(record as CustomChecksResponse)}
	>
		{#snippet item({ record, onSelect })}
			{@const integration = record as CustomChecksResponse}
			<ItemCard
				avatar={integration.logo
					? pb.files.getURL(integration, integration.logo)
					: undefined}
				title={integration.name}
				subtitle={ownerSubtitle(integration)}
				onClick={() => onSelect(record)}
			/>
		{/snippet}
	</StepCollectionPicker>
{/if}
