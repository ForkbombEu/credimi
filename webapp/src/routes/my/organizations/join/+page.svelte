<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { currentUser, pb } from '@/pocketbase/index.js';
	import {
		OrgJoinRequestsStatusOptions,
		type Data,
		type OrgJoinRequestsRecord,
		type OrganizationsResponse
	} from '@/pocketbase/types';
	import { m } from '@/i18n';
	import PageTop from '@/components/layout/pageTop.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import { ArrowLeft, Users, UserPlus } from 'lucide-svelte';
	import SectionTitle from '@/components/ui-custom/sectionTitle.svelte';
	import PageContent from '@/components/layout/pageContent.svelte';
	import PageCard from '@/components/layout/pageCard.svelte';
	import PlainCard from '@/components/ui-custom/itemCard.svelte';
	import { CollectionManager } from '@/collections-components';
	import Dialog from '@/components/ui-custom/dialog.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';

	//

	async function sendJoinRequest(org: OrganizationsResponse) {
		pb.collection('orgJoinRequests').create({
			user: $currentUser?.id!,
			organization: org.id!,
			status: OrgJoinRequestsStatusOptions.pending,
			reminders: 0
		} satisfies Data<OrgJoinRequestsRecord>);
	}
</script>

<PageTop>
	<Button href="/my/organizations" variant="outline" size="sm">
		<Icon src={ArrowLeft} mr></Icon>
		{m.Back_to_my_organizations()}
	</Button>

	<SectionTitle tag="h3" title={m.Join_an_organization()} hideLine></SectionTitle>
</PageTop>

<PageContent>
	<PageCard>
		<CollectionManager
			collection="organizations"
			queryOptions={{
				filter: `(id = orgAuthorizations_via_organization.organization.id && orgAuthorizations_via_organization.user.id != '${$currentUser?.id}')`,
				expand: ['orgJoinRequests_via_organization'],
				perPage: 20
			}}
			subscribe="expanded_collections"
			hide={['empty_state']}
		>
			{#snippet top({ Search })}
				<Search />
			{/snippet}

			{#snippet emptyState({ EmptyState })}
				<EmptyState title={m.No_available_organizations_found()} icon={Users} />
			{/snippet}

			{#snippet records({ records })}
				<div class="space-y-2">
					{#each records as org}
						{@const sentMembershipRequest =
							org.expand?.orgJoinRequests_via_organization?.at(0)}

						<PlainCard>
							{#snippet left()}
								<!-- TODO - Fetch info from organization_info -->
								<!-- <OrganizationAvatar organization={org} /> -->
							{/snippet}

							{#snippet children({ Title, Description })}
								<div>
									<Title>{org.name}</Title>
									<!-- TODO - Fetch info from organization_info -->
								</div>
							{/snippet}

							{#snippet right()}
								<div class="shrink-0 self-start pl-8">
									{#if !sentMembershipRequest}
										<Dialog title={`${m.Send_a_request_to()} ${org.name}`}>
											{#snippet trigger({ props })}
												<Button variant="outline" {...props}>
													{m.Join()}
													<Icon src={UserPlus} ml></Icon>
												</Button>
											{/snippet}

											{#snippet content({ closeDialog })}
												<T
													>{m.Please_confirm_that_you_want_to_join_this_organization_()}</T
												>
												<div
													class="flex items-center justify-center gap-2 pt-4"
												>
													<Button variant="outline" onclick={closeDialog}>
														{m.Cancel()}
													</Button>
													<Button
														onclick={() =>
															sendJoinRequest(org).then(closeDialog)}
													>
														{m.Send_join_request()}
													</Button>
												</div>
											{/snippet}
										</Dialog>
									{:else}
										<Button variant="outline" disabled
											>{m.Request_sent()}</Button
										>
									{/if}
								</div>
							{/snippet}
						</PlainCard>
					{/each}
				</div>
			{/snippet}
		</CollectionManager>
	</PageCard>
</PageContent>
