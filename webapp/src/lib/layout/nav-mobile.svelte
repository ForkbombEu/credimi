<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onNavigate } from '$app/navigation';
	import { Menu } from '@lucide/svelte';

	import type { LinkWithIcon } from '@/components/types';

	import { AppLogo } from '@/brand';
	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import * as Sheet from '@/components/ui/sheet';
	import { m } from '@/i18n';

	import NavLink from './nav-link.svelte';

	//

	interface Props {
		items: LinkWithIcon[];
	}

	const { items }: Props = $props();

	let open = $state(false);

	onNavigate(() => {
		open = false;
	});
</script>

<Sheet.Root bind:open>
	<Sheet.Trigger>
		{#snippet child({ props })}
			<Button variant="ghost" size="icon" class="lg:hidden" {...props}>
				<Icon src={Menu} />
			</Button>
		{/snippet}
	</Sheet.Trigger>

	<Sheet.Content side="right" class="w-80">
		<Sheet.Header class="border-b pb-4">
			<Sheet.Title>
				<AppLogo />
			</Sheet.Title>
		</Sheet.Header>

		<div class="mt-6 flex flex-col space-y-2">
			{#each items as item}
				<NavLink
					link={item}
					variant="mobile"
					badge={item.href?.endsWith('/my/tests/new') ? m.Beta() : undefined}
				/>
			{/each}
		</div>
	</Sheet.Content>
</Sheet.Root>
