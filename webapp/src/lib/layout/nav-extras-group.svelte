<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ChevronDownIcon } from '@lucide/svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import * as Collapsible from '@/components/ui/collapsible';
	import { m } from '@/i18n';

	import type { ExtraLink } from './topbar-links';

	import NavExtraLink from './nav-extra-link.svelte';

	interface Props {
		extras: ExtraLink[];
		onNavigate: () => void;
	}

	const { extras, onNavigate }: Props = $props();
</script>

<Collapsible.Root class="group/collapsible">
	<Collapsible.Trigger>
		{#snippet child({ props })}
			<Button {...props} variant="ghost" class="w-full justify-between">
				{m.Extras()}
				<ChevronDownIcon
					class="size-4 transition-transform group-data-[state=open]/collapsible:rotate-180"
				/>
			</Button>
		{/snippet}
	</Collapsible.Trigger>
	<Collapsible.Content class="mt-1 space-y-0.5">
		{#each extras as item (item.href)}
			<NavExtraLink link={item} onclick={onNavigate} />
		{/each}
	</Collapsible.Content>
</Collapsible.Root>
