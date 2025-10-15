<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { page } from '$app/state';
	import OrganizationPageDemo from '$lib/pages/organization-page.svelte';
	import { InfoIcon, Pencil, Undo } from 'lucide-svelte';

	import type { OrganizationsFormData } from '@/pocketbase/types';

	import { CollectionForm } from '@/collections-components/index.js';
	import { PageCard } from '@/components/layout/index.js';
	import Alert from '@/components/ui-custom/alert.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import * as AlertDialog from '@/components/ui/alert-dialog';
	import { m } from '@/i18n/index.js';

	//

	let { data } = $props();
	const { organization, marketplaceItems, isOrganizationNotEdited } = $derived(data);

	const isEdit = $derived(Boolean(page.url.searchParams.get('edit')));

	// svelte-ignore state_referenced_locally
	let showOrganizationForm = $state(isEdit);

	// Organization name change warning
	let showNameChangeWarning = $state(false);
	let pendingSubmitResolve: ((value: OrganizationsFormData) => void) | null = null;
	let pendingSubmitReject: ((reason?: unknown) => void) | null = null;
	let pendingData: OrganizationsFormData | null = null;

	async function handleBeforeSubmit(data: OrganizationsFormData): Promise<OrganizationsFormData> {
		const nameChanged = data.name !== organization?.name;
		if (!nameChanged) {
			return data;
		}
		return new Promise<OrganizationsFormData>((resolve, reject) => {
			pendingData = data;
			pendingSubmitResolve = resolve;
			pendingSubmitReject = reject;
			showNameChangeWarning = true;
		});
	}

	function confirmNameChange() {
		showNameChangeWarning = false;
		if (pendingData && pendingSubmitResolve) {
			pendingSubmitResolve(pendingData);
		}
		pendingSubmitResolve = null;
		pendingSubmitReject = null;
		pendingData = null;
	}

	function cancelNameChange() {
		showNameChangeWarning = false;
		if (pendingSubmitReject) {
			pendingSubmitReject(new Error('User cancelled'));
		}
		pendingSubmitResolve = null;
		pendingSubmitReject = null;
		pendingData = null;
	}
</script>

<AlertDialog.Root bind:open={showNameChangeWarning}>
	<AlertDialog.Content class="!z-[60]">
		<AlertDialog.Header>
			<AlertDialog.Title>{m.Change_Organization_Name()}</AlertDialog.Title>
			<AlertDialog.Description>
				<strong>{m.Warning()}:</strong>
				{m.Rename_organization_warning()}
			</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<Button variant="outline" onclick={cancelNameChange}>{m.Cancel()}</Button>
			<Button onclick={confirmNameChange}>{m.Continue()}</Button>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>

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
			beforeSubmit={handleBeforeSubmit}
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
