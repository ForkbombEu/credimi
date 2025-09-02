<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Pencil, Plus, X } from 'lucide-svelte';

	import { CollectionForm } from '@/collections-components';
	import { CollectionManager } from '@/collections-components/manager';
	import { RecordEdit, RecordDelete } from '@/collections-components/manager';
	import PageCard from '@/components/layout/pageCard.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import Dialog from '@/components/ui-custom/dialog.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import PlainCard from '@/components/ui-custom/itemCard.svelte';
	import SectionTitle from '@/components/ui-custom/sectionTitle.svelte';
	import UserAvatar from '@/components/ui-custom/userAvatar.svelte';
	import { createToggleStore } from '@/components/ui-custom/utils';
	import Badge from '@/components/ui/badge/badge.svelte';
	import { m } from '@/i18n';
	import { OrgRoles, ProtectedOrgUI } from '@/organizations';
	import OrganizationLayout from '@/organizations/components/organizationLayout.svelte';
	import { currentUser } from '@/pocketbase/index.js';
	import { getUserDisplayName } from '@/pocketbase/utils';
	import { capitalize } from '@/utils/other';

	import InviteMembersForm from './_partials/inviteMembersForm.svelte';
	import MembershipRequests from './_partials/membershipRequests.svelte';
	import PendingInvites from './_partials/pendingInvites.svelte';

	interface Props {
		//
		data: any;
	}

	let { data }: Props = $props();
	let organization = $derived(data.organization);
	let userRole = $derived(data.userRole);

	//

	const showInviteModal = createToggleStore(false);
</script>

<OrganizationLayout org={organization}>
	<ProtectedOrgUI orgId={organization.id} roles={['admin', 'owner']}>
		<MembershipRequests {organization} />
		<PendingInvites {organization} />
	</ProtectedOrgUI>

	<PageCard>
		<CollectionManager
			collection="orgAuthorizations"
			queryOptions={{
				expand: ['user', 'role'],
				filter: `organization.id="${organization.id}"`
			}}
			formFieldsOptions={{
				hide: { organization: organization.id },
				relations: {
					role: { mode: 'select', displayFields: ['name'] },
					user: { displayFields: ['name'] }
				}
			}}
			editFormFieldsOptions={{
				exclude: ['user', 'organization']
			}}
		>
			{#snippet top()}
				<SectionTitle title={m.Members()} description={m.members_description()}>
					{#snippet right()}
						<ProtectedOrgUI orgId={organization.id} roles={['admin', 'owner']}>
							<Dialog bind:open={$showInviteModal} title={m.invite_members()}>
								{#snippet trigger({ props })}
									<Button variant="outline" {...props}>
										<Plus size="20" class="mr-2" />
										{m.invite_members()}
									</Button>
								{/snippet}

								{#snippet content({ closeDialog })}
									<InviteMembersForm
										organizationId={organization.id}
										onSuccess={closeDialog}
										onCancel={closeDialog}
									/>
								{/snippet}
							</Dialog>
						</ProtectedOrgUI>
					{/snippet}
				</SectionTitle>
			{/snippet}

			{#snippet records({ records })}
				<div class="space-y-2">
					{#each records as record}
						{@const user = record.expand?.user}
						{@const role = record.expand?.role}
						{#if user && role && userRole}
							<PlainCard>
								<div class="flex items-center gap-4">
									<UserAvatar {user}></UserAvatar>
									<p>
										{getUserDisplayName(user)}
									</p>
									<div class="flex gap-1">
										{#if user.id == $currentUser?.id}
											<Badge>{m.You()}</Badge>
										{/if}
										{#if role.name != OrgRoles.MEMBER}
											<Badge variant="secondary"
												>{capitalize(role.name)}</Badge
											>
										{/if}
									</div>
								</div>

								{#snippet right()}
									<ProtectedOrgUI
										orgId={organization.id}
										roles={['admin', 'owner']}
									>
										<div class="space-x-1">
											{#if userRole.level < role.level}
												<RecordEdit {record}>
													{#snippet button({ triggerAttributes })}
														<Button
															variant="outline"
															size="sm"
															{...triggerAttributes}
														>
															{m.Edit_role()}
															<Icon src={Pencil} ml />
														</Button>
													{/snippet}
												</RecordEdit>

												<RecordDelete {record}>
													{#snippet button({ triggerAttributes })}
														<Button
															variant="outline"
															size="sm"
															{...triggerAttributes}
														>
															{m.Remove()}
															<Icon src={X} ml />
														</Button>
													{/snippet}
												</RecordDelete>
											{/if}
										</div>
									</ProtectedOrgUI>
								{/snippet}
							</PlainCard>
						{/if}
					{/each}
				</div>
			{/snippet}
		</CollectionManager>
	</PageCard>
</OrganizationLayout>
