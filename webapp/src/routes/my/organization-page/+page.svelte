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
	import { currentUser } from '@/pocketbase/index.js';
	import { InfoIcon, Plus, Pencil, Undo } from 'lucide-svelte';
	import OrganizationPageDemo from '$lib/pages/organization-page.svelte';

	let { data } = $props();
	const { organization, isOrganizationNotEdited } = $derived(data);

	//

	let showOrganizationForm = $state(false);
</script>

{#if !showOrganizationForm}
	<div class="mb-6 flex items-center justify-between">
		<T tag="h3">Page preview</T>
		<Button onclick={() => (showOrganizationForm = true)}>
			<Pencil />
			Edit organization info
		</Button>
	</div>
	<PageCard>
		<div class="overflow-hidden rounded-lg border">
			<OrganizationPageDemo {organization} />
		</div>
	</PageCard>
{:else}
	<div class="mb-6 flex items-center justify-between">
		<T tag="h3">
			{isOrganizationNotEdited
				? 'Update your organization page'
				: 'Create your organization page'}
		</T>

		<Button onclick={() => (showOrganizationForm = false)}>
			<Undo />
			Back
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
				{isOrganizationNotEdited ? 'Update organization page' : 'Create organization page'}
			{/snippet}
		</CollectionForm>
	</PageCard>
{/if}
