<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ComponentProps } from 'svelte';

	import type { UsersResponse } from '@/pocketbase/types';

	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import Avatar from './avatar.svelte';

	type Props = ComponentProps<typeof Avatar> & { user?: UsersResponse };

	let { user = pb.authStore.model as UsersResponse, ...rest }: Props = $props();
	if (!user) throw new Error('missing user');

	const src = $derived(pb.files.getURL(user, user.avatar));
	const fallback = $derived(user.name.slice(0, 2));
</script>

<Avatar {...rest} {src} {fallback} alt="{m.Avatar()} {user.name}" />
