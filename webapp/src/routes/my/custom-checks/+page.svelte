<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PublishedSwitch from '$lib/layout/published-switch.svelte';
	import { appSections } from '$lib/marketplace/sections';
	import { String } from 'effect';
	import { Pencil, Plus } from 'lucide-svelte';
	import removeMd from 'remove-markdown';

	import { CollectionManager } from '@/collections-components';
	import { RecordDelete } from '@/collections-components/manager';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import Card from '@/components/ui-custom/card.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Separator } from '@/components/ui/separator';
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
		{#snippet records({ records })}
			<div class="space-y-6">
				{#each records as record (record.id)}
					{@const logo = pb.files.getURL(record, record.logo)}
					{@const content = removeMd(record.description)}

					<Card class="bg-background" contentClass="p-4">
						<div class="flex items-center justify-between">
							<div class="flex items-center gap-4">
								<Avatar
									src={logo}
									class="rounded-sm"
									fallback={record.name.slice(0, 2)}
								/>
								<div>
									<T class="font-bold">{record.name}</T>
									<T class="mb-2 font-mono text-xs">
										{record.standard_and_version}
									</T>
								</div>
							</div>

							<div class="flex gap-1">
								<PublishedSwitch {record} field="public" />
								<IconButton
									href="/my/custom-checks/edit-{record.id}"
									icon={Pencil}
								/>
								<RecordDelete {record} />
							</div>
						</div>

						{#if String.isNonEmpty(content)}
							<div class="space-y-3 pt-3">
								<Separator />
								<T class="text-sm text-gray-400">{content}</T>
							</div>
						{/if}
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
