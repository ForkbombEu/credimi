<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import { afterNavigate } from '$app/navigation';

	const history = $state<URL[]>([]);

	export function storePreviousPageOnNavigate() {
		afterNavigate(({ from, type }) => {
			if (type === 'popstate') history.pop();
			else if (from) history.push(from.url);
		});
	}
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { ArrowLeft } from '@lucide/svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import { goto, m } from '@/i18n';

	//

	type Props = {
		href: string;
		children?: Snippet;
		class?: string;
	};

	let { href, children, class: className = '' }: Props = $props();

	storePreviousPageOnNavigate();

	function back() {
		const previousUrl = history.at(-1);
		if (!previousUrl) goto(href);
		else window.history.back();
	}
</script>

<Button onclick={back} variant="link" class={['gap-1 p-0', className]}>
	<ArrowLeft size={16} />
	{#if children}
		{@render children()}
	{:else}
		{m.Back()}
	{/if}
</Button>
