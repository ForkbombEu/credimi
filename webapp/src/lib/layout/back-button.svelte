<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import { afterNavigate } from '$app/navigation';

	const history = $state<URL[]>([]);

	const previousHref = $derived.by(() => {
		const last = history.at(-1);
		if (!last) return undefined;
		return last.pathname + last.search;
	});

	export function storePreviousPageOnNavigate() {
		afterNavigate(({ from, to, type }) => {
			if (type === 'popstate') history.pop();
			else if (!from?.url.pathname) return;
			else if (to && to.url.pathname === history.at(-1)?.pathname) history.pop();
			else history.push(from.url);
		});
	}
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { ArrowLeft } from '@lucide/svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import { m } from '@/i18n';

	//

	type Props = {
		href: string;
		children?: Snippet;
		class?: string;
	};

	let { href, children, class: className = '' }: Props = $props();

	storePreviousPageOnNavigate();

	const actualHref = $derived(previousHref ?? href);
</script>

<Button href={actualHref} variant="link" class={['gap-1 p-0', className]}>
	<ArrowLeft size={16} />
	{#if children}
		{@render children()}
	{:else}
		{m.Back()}
	{/if}
</Button>
