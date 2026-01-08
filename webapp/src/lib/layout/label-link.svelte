<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import A from '@/components/ui-custom/a.svelte';
	import CopyButtonSmall from '@/components/ui-custom/copy-button-small.svelte';
	import T from '@/components/ui-custom/t.svelte';

	interface Props {
		label: string;
		href?: string;
		published: boolean;
		class?: string;
		textToCopy?: string;
	}

	let { label, href, published, class: classValue, textToCopy }: Props = $props();
</script>

<T class={['flex items-center gap-1 font-medium', classValue]}>
	{#if published && href}
		<A class="underline underline-offset-2 hover:!no-underline" {href}>
			{label}
		</A>
	{:else}
		<span>
			{label}
		</span>
	{/if}
	{#if textToCopy}
		<CopyButtonSmall {textToCopy} square variant="ghost" size="xs" />
	{/if}
</T>
