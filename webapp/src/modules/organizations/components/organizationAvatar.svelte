<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Avatar, { type AvatarProps } from '@/components/ui-custom/avatar.svelte';
	import { cn } from '@/components/ui/utils';
	import { pb } from '@/pocketbase';
	import type { OrganizationInfoResponse, OrganizationsRecord } from '@/pocketbase/types';

	//

	type Props = AvatarProps & {
		organization: OrganizationsRecord;
	};

	let { organization, ...rest }: Props = $props();

	//

	let organizationInfoPromise = $derived(
		pb.collection('organization_info').getFirstListItem(`organization = "${organization.id}"`)
	);
</script>

{#await organizationInfoPromise then organizationInfo}
	{@const src = pb.files.getURL(organizationInfo, organizationInfo.logo ?? '')}
	{@const fallback = organization.name.slice(0, 2)}
	<Avatar {...rest} {src} {fallback} class={cn(rest.class, 'rounded-sm')} />
{/await}
