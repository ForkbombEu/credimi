<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { appSections } from '$lib/marketplace/sections';
	import { Pencil, Plus } from 'lucide-svelte';

	import { CollectionManager } from '@/collections-components';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import RenderMd from '@/components/ui-custom/renderMD.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import { setDashboardNavbar } from '../+layout@.svelte';

	let { data } = $props();
	const organizationId = $derived(data.organization.id);

	const { custom_checks } = appSections;
	setDashboardNavbar({ title: custom_checks.label, right: navbarRight });
</script>

<div class="space-y-4">
	<CollectionManager
		collection="custom_checks"
		queryOptions={{ filter: `owner.id = "${organizationId}"` }}
	>
		{#snippet records({ records, Card })}
			<div class="space-y-2">
				{#each records as record (record.id)}
					{@const logo = pb.files.getURL(record, record.logo)}
					<Card {record} class="bg-background !pl-4" hide={['share', 'select', 'edit']}>
						<div class="flex items-start gap-4">
							<Avatar
								src={logo}
								class="rounded-sm"
								fallback={record.name.slice(0, 2)}
							/>
							<div>
								<T class="font-bold">{record.name}</T>
								<T class="mb-2 font-mono text-xs">{record.standard_and_version}</T>
								<T class="text-sm text-gray-400"
									><RenderMd content={record.description}></RenderMd></T
								>
							</div>
						</div>

						{#snippet right()}
							<IconButton href="/my/custom-checks/edit-{record.id}" icon={Pencil} />
						{/snippet}
					</Card>
				{/each}
			</div>
		{/snippet}
	</CollectionManager>
</div>

{#snippet navbarRight()}
	<Button href="/my/custom-checks/new">
		<Plus />
		{m.Add_a_custom_check()}
	</Button>
{/snippet}
