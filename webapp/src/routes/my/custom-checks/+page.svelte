<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CollectionManager } from '@/collections-components';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';
	import T from '@/components/ui-custom/t.svelte';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import { Pencil, Plus } from 'lucide-svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import RenderMd from '@/components/ui-custom/renderMD.svelte';

	let { data } = $props();
	const organizationId = $derived(data.organization?.id);
</script>

<div class="space-y-4">
	<CollectionManager
		collection="custom_checks"
		queryOptions={{ filter: `owner.id = "${organizationId}"` }}
	>
		{#snippet top({ Header })}
			<Header title={m.Custom_checks()} hideCreate>
				{#snippet right()}
					<Button href="/my/custom-checks/new">
						<Plus />
						{m.Add_a_custom_check()}
					</Button>
				{/snippet}
			</Header>
		{/snippet}

		{#snippet records({ records, Card })}
			<div class="space-y-2">
				{#each records as record}
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
								<T class="text-sm text-gray-400"><RenderMd content={record.description}></RenderMd></T>
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
