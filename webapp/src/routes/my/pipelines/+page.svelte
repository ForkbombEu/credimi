<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import DashboardCard from '$lib/layout/dashboard-card.svelte';
	import { Plus } from 'lucide-svelte';

	import { CollectionManager } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import { setDashboardNavbar } from '../+layout@.svelte';

	let { data } = $props();
	let { organization } = $derived(data);

	setDashboardNavbar({ title: 'Pipelines', right: navbarRight });
</script>

<CollectionManager collection="pipelines">
	{#snippet records({ records })}
		<div class="space-y-6">
			{#each records as pipeline (pipeline.id)}
				<DashboardCard
					record={pipeline}
					avatar={() => pb.files.getURL(organization, organization.logo)}
				/>
			{/each}
		</div>
	{/snippet}
</CollectionManager>

{#snippet navbarRight()}
	<Button href="/my/pipelines/new">
		<Plus />
		{m.New()}
	</Button>
{/snippet}
