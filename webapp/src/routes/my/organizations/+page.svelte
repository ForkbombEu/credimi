<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { OrgRoles } from '@/organizations';
	import { currentUser, pb } from '@/pocketbase/index.js';
	import { invalidateAll } from '$app/navigation';
	import { m } from '@/i18n';
	import PlainCard from '@/components/ui-custom/itemCard.svelte';
	import CollectionManager from '@/collections-components/manager/collectionManager.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import { Badge } from '@/components/ui/badge';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import { PageTop, PageCard, PageContent } from '@/components/layout';
	import SectionTitle from '@/components/ui-custom/sectionTitle.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import { Plus, UserPlus, Cog, Puzzle, Undo2, X, Check } from 'lucide-svelte';
	import { capitalize } from '@/utils/other';
	import A from '@/components/ui-custom/a.svelte';
	import { toast } from 'svelte-sonner';
	import RenderMd from '@/components/ui-custom/renderMD.svelte';

	//

	const { ADMIN, OWNER } = OrgRoles;

	/* Org invites */

	function updateInvite(inviteId: string, action: 'accept' | 'decline') {
		pb.send(`/organizations/invites/${action}`, {
			method: 'POST',
			body: {
				inviteId
			}
		})
			.then(() => {
				if (action == 'accept') toast.success(m.Invite_accepted_succesfully());
				else if (action == 'decline') toast.info(m.Invitation_declined());
			})
			.catch(() => {
				toast.error(m.An_error_occurred_while_processing_your_request());
			});
	}

	/* Org membership requests */

	async function deleteJoinRequest(requestId: string) {
		await pb.collection('orgJoinRequests').delete(requestId);
		invalidateAll();
	}
</script>

<PageTop>
	<SectionTitle
		tag="h3"
		title={m.My_organizations()}
		description={m.organzations_page_description()}
	/>
</PageTop>

<PageContent>
	<CollectionManager
		collection="org_invites"
		queryOptions={{
			expand: ['organization'],
			filter: `user.id = "${$currentUser?.id ?? ''}" && declined = false`
		}}
		hide={['empty_state']}
	>
		{#snippet records({ records })}
			<PageCard>
				<SectionTitle title={m.organization_invites()} />

				{#each records as record}
					<PlainCard>
						{record.expand?.organization?.name}
						{#snippet right()}
							<Button
								variant="outline"
								onclick={() => updateInvite(record.id, 'accept')}
							>
								{m.accept_invite()}<Icon src={Check} ml />
							</Button>
							<Button
								variant="outline"
								onclick={() => updateInvite(record.id, 'decline')}
							>
								{m.decline_invite()}<Icon src={X} ml />
							</Button>
						{/snippet}
					</PlainCard>
				{/each}
			</PageCard>
		{/snippet}
	</CollectionManager>

	<CollectionManager
		collection="orgJoinRequests"
		queryOptions={{
			expand: ['organization'],
			filter: `user.id = "${$currentUser?.id}"`
		}}
		hide={['empty_state']}
	>
		{#snippet records({ records })}
			<PageCard>
				<SectionTitle title={m.Your_membership_requests()}></SectionTitle>

				<div class="space-y-4">
					{#each records as request}
						{@const organization = request.expand?.organization}
						{#if organization}
							{@const avatarUrl = pb.files.getURL(organization, organization.logo)}
							<PlainCard>
								{#snippet left()}
									<Avatar src={avatarUrl}></Avatar>
								{/snippet}

								{#snippet children({ Title })}
									<div class="flex items-center space-x-2">
										<Title>{request.expand?.organization?.name}</Title>
										<Badge variant="default">{m.Pending()}</Badge>
									</div>
								{/snippet}

								{#snippet right()}
									<Button
										variant="outline"
										onclick={() => {
											deleteJoinRequest(request.id);
										}}
									>
										{m.Undo_request()}
										<Icon src={Undo2} ml></Icon>
									</Button>
								{/snippet}
							</PlainCard>
						{/if}
					{/each}
				</div>
			</PageCard>
		{/snippet}
	</CollectionManager>

	<PageCard>
		<SectionTitle title={m.Your_organizations()}>
			{#snippet right()}
				<Button
					size="sm"
					variant="outline"
					class="shrink-0 !px-4"
					href="/my/organizations/join"
				>
					<span class="ml-1"> {m.Join_an_organization()} </span>
					<Icon src={UserPlus} ml />
				</Button>
				<Button size="sm" class="shrink-0 !px-4" href="/my/organizations/create">
					<span class="ml-1"> {m.Create_a_new_organization()} </span>
					<Icon src={Plus} ml />
				</Button>
			{/snippet}
		</SectionTitle>

		<CollectionManager
			collection="orgAuthorizations"
			queryOptions={{
				expand: ['organization', 'role'],
				filter: `user.id = "${$currentUser?.id}"`
			}}
		>
			{#snippet emptyState({ EmptyState })}
				<EmptyState title={m.You_havent_added_any_organizations_yet_()} icon={Puzzle} />
			{/snippet}

			{#snippet records({ records })}
				<div class="space-y-2">
					{#each records as a}
						{@const org = a.expand?.organization}
						{@const role = a.expand?.role}
						{#if org && role}
							<PlainCard>
								{#snippet children({ Title, Description })}
									<div class="flex items-center gap-2">
										<Title>
											<A href={`/my/organizations/${org.id}`}>{org.name}</A>
										</Title>
										{#if role.name == ADMIN || role.name == OWNER}
											<Badge variant="secondary"
												>{capitalize(role.name)}</Badge
											>
										{/if}
									</div>
									{#if org.description}
										<Description><RenderMd content={org.description}/></Description>
									{/if}
								{/snippet}
								{#snippet right()}
									{#if role.name == OWNER}
										<Button
											data-testid={`${org.name} link`}
											size="sm"
											variant="outline"
											href={`/my/organizations/${org.id}/settings`}
										>
											{m.Settings()}
											<Icon src={Cog} ml></Icon>
										</Button>
									{/if}
								{/snippet}
							</PlainCard>
						{/if}
					{/each}
				</div>
			{/snippet}
		</CollectionManager>
	</PageCard>
</PageContent>
