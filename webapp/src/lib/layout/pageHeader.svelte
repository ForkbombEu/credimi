<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { ArrowUpRight } from '@lucide/svelte';

	import type { SnippetFunction } from '@/components/types';

	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import T from '@/components/ui-custom/t.svelte';

	interface Props {
		id: string;
		class?: string;
		title?: string;
		children?: Snippet;
		right?: Snippet<[{ Link: SnippetFunction<LinkProps> }]>;
	}

	interface LinkProps {
		href?: string;
		label: string;
		onclick?: () => void;
	}

	let { title, id, class: className = '', children, right }: Props = $props();
</script>

<div
	{id}
	class={[
		'border-secondary-foreground mb-6 flex scroll-mt-5 items-center justify-between border-b',
		className
	]}
>
	{#if title}
		<T tag="h2">{title}:</T>
	{/if}

	{@render children?.()}

	{@render right?.({ Link })}
</div>

{#snippet Link(props: LinkProps)}
	{@const { href, label, onclick } = props}
	<Button variant="link" {href} {onclick} class="gap-1 underline hover:no-underline">
		<T>{label}</T>
		<Icon src={ArrowUpRight} />
	</Button>
{/snippet}
