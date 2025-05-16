<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { CollectionForm } from '@/collections-components/index.js';
	import { PageCard } from '@/components/layout/index.js';
	import Alert from '@/components/ui-custom/alert.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { InfoIcon, Pencil, Undo } from 'lucide-svelte';
	import OrganizationPageDemo from '$lib/pages/organization-page.svelte';
	import { page } from '$app/state';
	import { m } from '@/i18n/index.js';

	//

	let { data } = $props();
	const { organization, marketplaceItems, isOrganizationNotEdited } = $derived(data);

	const isEdit = $derived(Boolean(page.url.searchParams.get('edit')));

	// svelte-ignore state_referenced_locally
	let showOrganizationForm = $state(isEdit);
</script>

{#if isOrganizationNotEdited}
	<Alert variant="info" icon={InfoIcon} class="mb-8">
		<T>
			{m.Edit_your_organization_information_to_better_represent_your_services_and_products_on_the_marketplace()}
		</T>
	</Alert>
{/if}

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
		<!-- TODO - Set owner via hook -->
		<CollectionForm
			collection="organizations"
			onSuccess={() => {
				invalidateAll();
				showOrganizationForm = false;
			}}
			initialData={organization}
			recordId={organization?.id}
		>
			{#snippet submitButtonContent()}
				{m.Update_organization_page()}
			{/snippet}
		</CollectionForm>
	</PageCard>
{/if}
