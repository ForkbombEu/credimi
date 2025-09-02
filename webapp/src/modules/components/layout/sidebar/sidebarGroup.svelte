<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import * as Collapsible from '@/components/ui/collapsible';
	import * as Sidebar from '@/components/ui/sidebar';
	import { cn } from '@/components/ui/utils';

	type Props = { trigger: Snippet; content: Snippet; active?: boolean };

	const { trigger, content, active = false }: Props = $props();

	const open = $derived(active);
</script>

<Sidebar.Menu class={cn({ 'ring-primary rounded-sm ring-1 ring-offset-2': active })}>
	<Collapsible.Root class="group/collapsible" {open}>
		<Sidebar.MenuItem>
			<Collapsible.Trigger>
				{#snippet child({ props })}
					<Sidebar.MenuButton {...props}>
						{@render trigger()}
					</Sidebar.MenuButton>
				{/snippet}
			</Collapsible.Trigger>
			<Collapsible.Content>
				<Sidebar.MenuSub>
					{@render content()}
				</Sidebar.MenuSub>
			</Collapsible.Content>
		</Sidebar.MenuItem>
	</Collapsible.Root>
</Sidebar.Menu>
