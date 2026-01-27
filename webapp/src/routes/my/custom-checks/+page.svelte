<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Pencil, PlayIcon, Plus } from '@lucide/svelte';
	import { entities } from '$lib/global/entities';
	import DashboardCard from '$lib/layout/dashboard-card.svelte';
	import { getCustomCheckPublicUrl } from '$lib/marketplace/utils.js';

	import { CollectionManager } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import { setDashboardNavbar } from '../+layout@.svelte';

	//

	let { data } = $props();
	const organizationId = $derived(data.organization.id);

	setDashboardNavbar({ title: entities.custom_checks.labels.plural, right: navbarRight });
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
				>
					{#snippet actions()}
						<Button href={getCustomCheckPublicUrl(record)}>
							<PlayIcon />
							{m.Run_now()}
						</Button>
					{/snippet}
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
