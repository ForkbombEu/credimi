<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import Avatar from '@/components/ui-custom/avatar.svelte';
	import CopyButtonSmall from '@/components/ui-custom/copy-button-small.svelte';
	import T from '@/components/ui-custom/t.svelte';
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
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
		<a href={localizeHref(href)} class="hover:underline">
			<T class="overflow-hidden text-ellipsis font-semibold">{name}</T>
		</a>

		{#if textToCopy}
			<CopyButtonSmall {textToCopy} square variant="ghost" size="xs" />
		{/if}
	</div>

	{@render children?.()}
</div>
