<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowUpRight, InfoIcon } from 'lucide-svelte';

	import { CollectionForm } from '@/collections-components/index.js';
	import Alert from '@/components/ui-custom/alert.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n/index.js';

	//

	let { data } = $props();
	let { organization, isOrganizationNotEdited } = $derived(data);
</script>

<div class="flex items-center justify-between">
	<T tag="h3">{m.Update_your_organization_page()}</T>
	<Button variant="outline" href="/organizations/{organization.canonified_name}">
		{m.Page_preview()}
		<ArrowUpRight />
	</Button>
</div>

{#if isOrganizationNotEdited}
	<Alert variant="info" icon={InfoIcon} class="mb-8">
		<T>
			{m.Edit_your_organization_information_to_better_represent_your_services_and_products_on_the_marketplace()}
		</T>
	</Alert>
{/if}

{#key organization}
	<CollectionForm
		collection="organizations"
		initialData={organization}
		recordId={organization.id}
		onSuccess={(org) => {
			organization = org;
		}}
		fieldsOptions={{
			exclude: ['canonified_name']
		}}
	>
		{#snippet submitButtonContent()}
			{m.Update_organization_page()}
		{/snippet}
	</CollectionForm>
{/key}

<!-- 
{#if !showOrganizationForm}
	<div class="mb-6 flex items-center justify-between">
		<T tag="h3">{m.Page_preview()}</T>
		<Button onclick={() => (showOrganizationForm = true)}>
			<Pencil />
			{m.Edit_organization_info()}
		</Button>
	</div>
	<PageCard contentClass="!p-2">
		<div class="overflow-hidden rounded-lg border">
			<OrganizationPageDemo organization={organization!} {marketplaceItems} isPreview />
		</div>
	</PageCard>
{:else}
	<div class="mb-6 flex items-center justify-between">
		<T tag="h3">{m.Update_your_organization_page()}</T>

		<Button onclick={() => (showOrganizationForm = false)}>
			<Undo />
			{m.Back()}
		</Button>
	</div>

	<PageCard>
		<CollectionForm
			collection="organizations"
			onSuccess={() => {
				invalidateAll();
				showOrganizationForm = false;
			}}
			initialData={organization}
			recordId={organization?.id}
			fieldsOptions={{
				exclude: ['canonified_name']
			}}
		>
			{#snippet submitButtonContent()}
				{m.Update_organization_page()}
			{/snippet}
		</CollectionForm>
	</PageCard>
{/if}
-->
