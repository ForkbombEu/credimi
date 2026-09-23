<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { resolve } from '$app/paths';

	import Avatar from '@/components/ui-custom/avatar.svelte';
	import CopyButtonSmall from '@/components/ui-custom/copy-button-small.svelte';
	import { localizeHref } from '@/i18n';

	//

	type Props = {
		logo?: string;
		name: string;
		/** Muted secondary line under the name (e.g. suite provider). */
		subtitle?: string;
		textToCopy?: string;
		href: string;
		children?: Snippet;
	};

	let { logo, name, subtitle, textToCopy, href, children }: Props = $props();
</script>

<div class="flex items-center gap-3">
	<Avatar src={logo ?? ''} class="size-10 rounded-sm! border" fallback={name.slice(0, 2)} />

	<div class="flex min-w-0 flex-col gap-0.5">
		<div class="flex items-center gap-1">
			<a href={resolve(localizeHref(href) as '/')} class="font-semibold hover:underline">
				{name}
			</a>

			{#if textToCopy}
				<CopyButtonSmall {textToCopy} />
			{/if}
		</div>

		{#if subtitle}
			<span class="text-xs text-muted-foreground">{subtitle}</span>
		{/if}
	</div>

	{@render children?.()}
</div>
