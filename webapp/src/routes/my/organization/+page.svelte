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
	import { InfoIcon, Plus, Pencil, Undo } from 'lucide-svelte';
	import OrganizationPageDemo from '$lib/pages/organization-page.svelte';
	import { page } from '$app/state';

	//
	let { data } = $props();
	const { organizationInfo, isOrganizationInfoMissing } = $derived(data);

	const isEdit = $derived(Boolean(page.url.searchParams.get('edit')));

	// svelte-ignore state_referenced_locally
	let showOrganizationForm = $state(isEdit);
</script>

{#if isOrganizationInfoMissing}
	<Alert variant="info" icon={InfoIcon} class="mb-8">
		<T>
			Edit your organization's information to better represent your services and products on
			the marketplace
		</T>
	</Alert>
{/if}

{#if !showOrganizationForm}
	{#if organizationInfo}
		<div class="mb-6 flex items-center justify-between">
			<T tag="h3">Page preview</T>
			<Button onclick={() => (showOrganizationForm = true)}>
				<Pencil />
				Edit organization info
			</Button>
		</div>
		<PageCard>
			<div class="overflow-hidden rounded-lg border">
				<OrganizationPageDemo {organizationInfo} />
			</div>
		</PageCard>
	{:else}
		<Alert variant="info" icon={InfoIcon}>
			{#snippet content({ Title, Description })}
				<Title>Info</Title>
				<Description class="mt-2">
					An organization page is used to present your services and products on the
					marketplace. Create one to get started!
				</Description>
				<div class="mt-2 flex justify-end">
					<Button
						variant="outline"
						onclick={() => {
							showOrganizationForm = true;
						}}
					>
						<Plus />
						Create organization page
					</Button>
				</div>
			{/snippet}
		</Alert>
	{/if}
{:else}
	<div class="mb-6 flex items-center justify-between">
		<T tag="h3">
			{organizationInfo ? 'Update your organization page' : 'Create your organization page'}
		</T>

		<Button onclick={() => (showOrganizationForm = false)}>
			<Undo />
			Back
		</Button>
	</div>

	<PageCard>
		<!-- TODO - Set owner via hook -->
		<CollectionForm
			collection="organization_info"
			fieldsOptions={{ exclude: ['owner'] }}
			onSuccess={() => {
				invalidateAll();
				showOrganizationForm = false;
			}}
			initialData={organizationInfo}
			recordId={organizationInfo?.id}
		>
			{#snippet submitButtonContent()}
				{organizationInfo ? 'Update organization page' : 'Create organization page'}
			{/snippet}
		</CollectionForm>
	</PageCard>
{/if}
