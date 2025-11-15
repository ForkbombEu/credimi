<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import DashboardCard from '$lib/layout/dashboard-card.svelte';
	import { appSections } from '$lib/marketplace/sections';
	import { Pencil, Plus } from 'lucide-svelte';

	import { CollectionManager } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import { setDashboardNavbar } from '../+layout@.svelte';

	//

	let { data } = $props();
	const organizationId = $derived(data.organization.id);

	const { custom_checks } = appSections;
	setDashboardNavbar({ title: custom_checks.label, right: navbarRight });
</script>

<CollectionManager
	collection="custom_checks"
	queryOptions={{ filter: `owner.id = "${organizationId}"` }}
>
	{#snippet records({ records })}
		<div class="space-y-6">
			{#each records as record (record.id)}
				<DashboardCard
					{record}
					avatar={(r) => pb.files.getURL(r, r.logo)}
					subtitle={record.standard_and_version}
					path={[data.organization.canonified_name, record.canonified_name]}
				>
					{#snippet editAction()}
						<IconButton href="/my/custom-checks/edit-{record.id}" icon={Pencil} />
					{/snippet}
				</DashboardCard>
			{/each}
		</div>
	{/snippet}
</CollectionManager>

{#snippet navbarRight()}
	<Button href="/my/custom-checks/new">
		<Plus />
		{m.Add_a_custom_check()}
	</Button>
{/snippet}
