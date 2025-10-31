<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { ClipboardPlus } from 'lucide-svelte';

	import Icon from '@/components/ui-custom/icon.svelte';
	import Button, { type ButtonProps } from '@/components/ui/button/button.svelte';
	import { m } from '@/i18n';

	import type { IconComponent } from '../types';

	//

	type Props = ButtonProps & {
		delay?: number;
		textToCopy: string;
		children?: Snippet;
		icon?: IconComponent;
	};

	let { textToCopy, delay = 2000, children, icon, ...rest }: Props = $props();

	let isCopied = $state(false);

	function copyText() {
		navigator.clipboard.writeText(textToCopy);
		isCopied = true;
		setTimeout(() => {
			isCopied = false;
		}, delay);
	}
</script>

<Button variant="outline" {...rest} onclick={copyText}>
	{#if !isCopied}
		{@render children?.()}
		<Icon src={icon ?? ClipboardPlus} ml={Boolean(children)} />
	{:else}
		<span class="whitespace-nowrap">✅ {m.Copied()}</span>
	{/if}
</Button>
