<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAnchorAttributes } from 'svelte/elements';

	import { resolve } from '$app/paths';

	import { localizeHref } from '@/i18n';

	//

	type Props = HTMLAnchorAttributes & {
		children?: Snippet;
	};

	let { class: className, href, children, ...rest }: Props = $props();
</script>

{#if href}
	<a
		href={resolve(localizeHref(href) as '/')}
		class={['text-primary hover:underline', className]}
		{...rest}
	>
		{@render children?.()}
	</a>
{:else}
	<p>
		{@render children?.()}
	</p>
{/if}
