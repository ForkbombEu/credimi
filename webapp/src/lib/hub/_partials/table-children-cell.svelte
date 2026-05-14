<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowRight } from '@lucide/svelte';
	import { resolve } from '$app/paths';

	import type { Link } from '@/components/types';

	import { localizeHref } from '@/i18n';

	//

	type Props = {
		items: Link[];
	};

	let { items }: Props = $props();
</script>

<div class="grid min-w-[400px] grid-cols-2 md:grid-cols-3">
	{#each items.filter((item) => item.title) as link (link)}
		<a
			href={resolve(localizeHref(link.href ?? '') as '/')}
			class={[
				'rounded-full px-2 py-1 text-nowrap text-slate-500 transition hover:bg-slate-300',
				'flex flex-nowrap items-center gap-1 text-xs'
			]}
		>
			<span class="truncate">{link.title}</span>
			<ArrowRight size={10} class="shrink-0" />
		</a>
	{/each}
</div>
