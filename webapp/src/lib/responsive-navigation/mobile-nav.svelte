<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import * as Sheet from '@/components/ui/sheet';
	import { m } from '@/i18n';
	import { Menu } from 'lucide-svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import type { Snippet } from 'svelte';

	interface Props {
		open: boolean;
		onOpenChange: (open: boolean) => void;
		children: Snippet;
		title?: string;
	}

	const { open, onOpenChange, children, title = m.navigation() }: Props = $props();
</script>

<Sheet.Root {open} {onOpenChange}>
	<Sheet.Trigger>
		{#snippet child({ props })}
			<Button variant="ghost" size="icon" class="lg:hidden" {...props}>
				<Icon src={Menu} />
			</Button>
		{/snippet}
	</Sheet.Trigger>

	<Sheet.Content side="left" class="w-80">
		<Sheet.Header class="border-b pb-4">
			<Sheet.Title>{title}</Sheet.Title>
		</Sheet.Header>

		<div class="mt-6 flex flex-col space-y-2">
			{@render children()}
		</div>
	</Sheet.Content>
</Sheet.Root>
