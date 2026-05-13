<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { resolve } from '$app/paths';

	import { localizeHref } from '@/i18n';

	import type { ChildLink } from './types';

	//

	type Props = {
		children: ChildLink[];
	};

	let { children }: Props = $props();
</script>

<ul class="flex list-disc flex-col">
	{#each children as child (child.href)}
		<li class="max-w-[35ch] truncate text-xs">
			<a class="text-primary hover:underline" href={resolve(localizeHref(child.href) as '/')}>
				{child.label}
			</a>
		</li>
	{/each}
</ul>
