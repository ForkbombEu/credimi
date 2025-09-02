<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { OrganizationsRecord } from '@/pocketbase/types';

	import Avatar, { type AvatarProps } from '@/components/ui-custom/avatar.svelte';
	import { cn } from '@/components/ui/utils';
	import { pb } from '@/pocketbase';

	type Props = AvatarProps & {
		organization: OrganizationsRecord;
	};

	let { organization, ...rest }: Props = $props();

	let src = $derived(pb.files.getURL(organization, organization.logo ?? ''));
	let fallback = $derived(organization.name.slice(0, 2));
</script>

<Avatar {...rest} {src} {fallback} class={cn(rest.class, 'rounded-sm')} />
