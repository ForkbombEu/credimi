<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CollectionManager } from '@/collections-components';
	import { pb } from '@/pocketbase/index.js';
	import {
		OrgJoinRequestsStatusOptions,
		type OrgJoinRequestsRecord,
		type OrgJoinRequestsResponse,
		type OrganizationsResponse
	} from '@/pocketbase/types';
	import { m } from '@/i18n';
	import { UserPlus, CircleOffIcon } from 'lucide-svelte';
	import PlainCard from '@/components/ui-custom/itemCard.svelte';
	import { getUserDisplayName } from '@/pocketbase/utils';
	import UserAvatar from '@/components/ui-custom/userAvatar.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import SectionTitle from '@/components/ui-custom/sectionTitle.svelte';
	import { PageCard } from '@/components/layout';
	import Button from '@/components/ui-custom/button.svelte';
	import Dialog from '@/components/ui-custom/dialog.svelte';
	import { toast } from 'svelte-sonner';
	import Trash from 'lucide-svelte/icons/trash';
	import X from 'lucide-svelte/icons/x';

	interface Props {
		//
		organization: OrganizationsResponse;
	}

	let { organization }: Props = $props();

	//

	const { accepted, rejected, pending } = OrgJoinRequestsStatusOptions;

	async function updateRequestStatus(
		request: OrgJoinRequestsResponse,
		status: OrgJoinRequestsStatusOptions
	) {
		await pb.collection('orgJoinRequests').update(request.id, {
			status
		} satisfies Partial<OrgJoinRequestsRecord>);
	}
</script>

<CollectionManager
	collection="orgJoinRequests"
	queryOptions={{
		filter: `organization.id = "${organization.id}" && status = "${pending}"`,
		expand: ['user']
	}}
	hide={['empty_state']}
>
	{#snippet records({ records })}
		<PageCard>
			<SectionTitle
				tag="h4"
				title={m.Pending_membership_requests()}
				description={m.pending_membership_requests_description()}
			/>

			<div class="space-y-2">
				{#each records as request}
					{@const user = request.expand?.user}
					{#if user}
						<PlainCard>
							{#snippet left()}
								<UserAvatar {user} />
							{/snippet}

							{getUserDisplayName(user)}

							{#snippet right()}
								<div class="space-x-1">
									<Button
										variant="outline"
										onclick={() =>
											updateRequestStatus(request, accepted)
												.then(() =>
													toast.success(m.Request_accepted_sucessfully())
												)
												.catch(() =>
													toast.error(
														m.An_error_occurred_while_handling_membership_request()
													)
												)}
									>
										{m.Accept()}
										<Icon src={UserPlus} ml />
									</Button>

									<Dialog title={m.Warning()}>
										{#snippet trigger({ props })}
											<Button variant="outline" {...props}>
												{m.Decline()}
												<Icon src={CircleOffIcon} ml />
											</Button>
										{/snippet}

										{#snippet content({ closeDialog })}
											<p>{m.decline_membership_request_warning()}</p>
											<div class="flex items-center justify-center gap-2">
												<Button variant="outline" onclick={closeDialog}>
													{m.Cancel()}
													<Icon src={X} ml />
												</Button>
												<Button
													variant="destructive"
													onclick={() =>
														updateRequestStatus(request, rejected)
															.then(closeDialog)
															.then(() =>
																toast.info(
																	m.Membership_request_declined_successfully()
																)
															)}
												>
													{m.decline_membership_request()}
													<Icon src={Trash} ml />
												</Button>
											</div>
										{/snippet}
									</Dialog>
								</div>
							{/snippet}
						</PlainCard>
					{/if}
				{/each}
			</div>
		</PageCard>
	{/snippet}
</CollectionManager>
