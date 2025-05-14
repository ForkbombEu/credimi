<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { CollectionForm } from '@/collections-components';
	import { m } from '@/i18n';
	import OrganizationLayout from '@/organizations/components/organizationLayout.svelte';
	import PageCard from '@/components/layout/pageCard.svelte';
	import T from '@/components/ui-custom/t.svelte';

	interface Props {
		//
		data: any;
	}

	let { data }: Props = $props();
	let organization = $derived(data.organization);
</script>

<OrganizationLayout org={organization}>
	<PageCard>
		<T tag="h4" class="mb-6">{m.Manage_your_organization_public_info()}</T>
		<CollectionForm
			collection="organizations"
			recordId={organization.id}
			initialData={organization}
			onSuccess={invalidateAll}
		>
			{#snippet submitButtonContent()}
				{m.Save_changes()}
			{/snippet}
		</CollectionForm>
	</PageCard>
</OrganizationLayout>
