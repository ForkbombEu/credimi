<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { LogOut } from '@lucide/svelte';
	import { fromStore } from 'svelte/store';

	import Button from '@/components/ui-custom/button.svelte';
	import DropdownMenuLink from '@/components/ui-custom/dropdownMenuLink.svelte';
	import UserAvatar from '@/components/ui-custom/userAvatar.svelte';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';
	import { m } from '@/i18n';
	import { currentUser } from '@/pocketbase';

	//

	const userState = fromStore(currentUser);
	const user = $derived(userState.current);
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Button variant="ghost" {...props} class="relative size-10 rounded-full border p-1">
				{#if user}
					<UserAvatar {user} class="size-8" />
				{/if}
			</Button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content class="w-56" align="end">
		{#if user}
			<DropdownMenu.Label class="font-normal">
				<div class="flex flex-col space-y-1">
					<p class="text-sm font-medium leading-none">{user.name}</p>
					<p class="text-muted-foreground text-xs leading-none">{user.email}</p>
				</div>
			</DropdownMenu.Label>
		{/if}
		<DropdownMenu.Separator />

		<DropdownMenu.Group>
			<DropdownMenuLink href="/my">
				{m.Go_to_Dashboard()}
			</DropdownMenuLink>
			<DropdownMenuLink href="/my/profile">
				{m.My_profile()}
			</DropdownMenuLink>
		</DropdownMenu.Group>

		<DropdownMenu.Separator />

		<DropdownMenuLink href="/logout" icon={LogOut}>
			{m.Sign_out()}
		</DropdownMenuLink>
	</DropdownMenu.Content>
</DropdownMenu.Root>
