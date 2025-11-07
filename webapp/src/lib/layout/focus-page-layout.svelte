<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import clsx from 'clsx';

	import type { Link } from '@/components/types';

	import T from '@/components/ui-custom/t.svelte';

	import BackButton from './back-button.svelte';

	//

	type Props = {
		title?: string;
		description?: string;
		top?: Snippet;
		class?: string;
		backButton?: Link;
		children: Snippet;
	};

	let { title, description, top, class: className, backButton, children }: Props = $props();
</script>

<div class={['bg-secondary min-h-screen space-y-10 p-6', className]}>
	{#if backButton}
		{@const { href, title, class: classes, ...rest } = backButton}
		{#if href}
			<BackButton {href} class={clsx(classes)} {...rest}>
				{title}
			</BackButton>
		{/if}
	{/if}

	{#if title || description || top}
		<div class="space-y-2">
			{#if title}
				<T tag="h1" class="text-center !text-3xl !font-semibold">
					{title}
				</T>
			{/if}
			{#if description}
				<T tag="p" class="text-center">
					<!-- eslint-disable-next-line svelte/no-at-html-tags -->
					{@html description}
				</T>
			{/if}

			{@render top?.()}
		</div>
	{/if}

	{@render children()}
</div>
