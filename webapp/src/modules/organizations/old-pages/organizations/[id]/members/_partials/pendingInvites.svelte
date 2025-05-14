<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import CollectionManager from '@/collections-components/manager/collectionManager.svelte';
	import type { OrganizationsResponse } from '@/pocketbase/types';
	import { m } from '@/i18n';
	import Badge from '@/components/ui/badge/badge.svelte';
	import SectionTitle from '@/components/ui-custom/sectionTitle.svelte';
	import { RecordDelete } from '@/collections-components/manager';
	import { PageCard } from '@/components/layout';
	import IconButton from '@/components/ui-custom/iconButton.svelte';

	interface Props {
		//
		organization: OrganizationsResponse;
	}

	let { organization }: Props = $props();
</script>

<CollectionManager
	collection="org_invites"
	queryOptions={{
		filter: `organization.id = "${organization.id}"`
	}}
	hide={['empty_state']}
>
	{#snippet records({ records })}
		<PageCard class="!space-y-6">
			<SectionTitle
				tag="h4"
				title={m.Pending_invites()}
				description={m.pending_invites_description()}
			/>

			<div class="max-h-[400px] divide-y overflow-auto rounded-lg border">
				{#each records as invite}
					<div class="flex items-center justify-between px-2 py-1 pl-4">
						<div class="flex items-center gap-2">
							<p class="text-sm">
								{invite.user_email}
							</p>
							{#if invite.failed_email_send}
								<Badge variant="destructive">{m.failed_email_send()}</Badge>
							{/if}
							{#if invite.declined}
								<Badge variant="destructive">{m.invite_declined()}</Badge>
							{/if}
						</div>

						<RecordDelete record={invite}>
							{#snippet button({ icon, triggerAttributes })}
								<IconButton variant="ghost" {icon} {...triggerAttributes} />
							{/snippet}
						</RecordDelete>
					</div>
				{/each}
			</div>
		</PageCard>
	{/snippet}
</CollectionManager>
