<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { resolve } from '$app/paths';

	import type { Link } from '@/components/types';

	import { localizeHref } from '@/i18n';

	//

	type NestedLink = Link & { dimmed?: boolean };

	type Props = {
		items: NestedLink[];
	};

	let { items }: Props = $props();
</script>

<div class="grid min-w-[400px] grid-cols-2 gap-x-2 gap-y-1 md:grid-cols-3">
	{#each items.filter((item) => item.title) as link (link)}
		<a
			href={resolve(localizeHref(link.href ?? '') as '/')}
			class={['truncate text-xs text-primary hover:underline', link.dimmed && 'opacity-40']}
		>
			{link.title}
		</a>
	{/each}
</div>
