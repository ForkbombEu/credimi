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
		textToCopy?: string;
		href: string;
		children?: Snippet;
	};

	let { logo, name, textToCopy, href, children }: Props = $props();
</script>

<div class="flex items-center gap-3">
	<Avatar src={logo ?? ''} class="size-10 rounded-sm! border" fallback={name.slice(0, 2)} />

	<div class="flex items-center gap-1">
		<a href={resolve(localizeHref(href) as '/')} class="font-semibold hover:underline">
			{name}
		</a>

		{#if textToCopy}
			<CopyButtonSmall {textToCopy} />
		{/if}
	</div>

	{@render children?.()}
</div>
