<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { MouseEventHandler } from 'svelte/elements';

	import { SquareArrowOutUpRight } from '@lucide/svelte';

	import type { ExtraLink } from './topbar-links';

	interface Props {
		link: ExtraLink;
		onclick?: MouseEventHandler<HTMLAnchorElement>;
	}

	const { link, onclick }: Props = $props();
</script>

<a
	href={link.href}
	target="_blank"
	rel="noopener noreferrer"
	class="block cursor-pointer space-y-0.5 rounded-md px-3 py-2 text-sm hover:bg-secondary/50"
	{onclick}
>
	<p class="font-medium text-primary">
		{link.title}
		<SquareArrowOutUpRight class="inline-block size-3 -translate-y-px" />
	</p>
	<p class="text-xs text-balance text-muted-foreground">
		{link.description}
	</p>
</a>
