<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import A from '@/components/ui-custom/a.svelte';
	import T from '@/components/ui-custom/t.svelte';

	interface Props {
		label: string;
		href: string;
		published: boolean;
		class?: string;
	}

	let { label, href, published, class: classValue }: Props = $props();
</script>

<T class={['font-medium', classValue]}>
	{#if !published}
		{label}
	{:else}
		<A class="underline underline-offset-2 hover:!no-underline" {href}>
			{label}
		</A>
	{/if}
</T>
