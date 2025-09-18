<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { CheckIcon, CopyIcon } from 'lucide-svelte';

	import Button, { type ButtonProps } from '@/components/ui/button/button.svelte';

	//

	type Props = ButtonProps & {
		delay?: number;
		textToCopy: string;
		children?: Snippet;
		square?: boolean;
	};

	let { textToCopy, delay = 1000, children, square = false, ...rest }: Props = $props();

	let isCopied = $state(false);

	function copyText() {
		navigator.clipboard.writeText(textToCopy);
		isCopied = true;
		setTimeout(() => {
			isCopied = false;
		}, delay);
	}
</script>

<Button
	variant="outline"
	class={[
		'h-8',
		{
			'w-8 p-0': square
		}
	]}
	{...rest}
	onclick={copyText}
>
	{#if !isCopied}
		<CopyIcon />
	{:else}
		<CheckIcon class="text-green-600" />
	{/if}

	{@render children?.()}
</Button>
