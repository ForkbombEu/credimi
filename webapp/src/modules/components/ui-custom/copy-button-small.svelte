<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ComponentProps, Snippet } from 'svelte';

	import { CheckIcon, CopyIcon } from 'lucide-svelte';

	import Button, { type ButtonProps } from '@/components/ui/button/button.svelte';

	import Icon from './icon.svelte';
	//

	type Props = Omit<ButtonProps, 'size'> & {
		delay?: number;
		textToCopy: string;
		children?: Snippet;
		square?: boolean;
		variant?: ComponentProps<typeof Button>['variant'];
		size?: 'sm' | 'xs';
	};

	let {
		textToCopy,
		delay = 1000,
		children,
		square = false,
		variant = 'outline',
		size = 'sm',
		...rest
	}: Props = $props();

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
	{variant}
	class={{
		'h-8': size == 'sm',
		'h-6': size == 'xs',
		'p-0': square,
		'w-8': square && size == 'sm',
		'w-6': square && size == 'xs'
	}}
	{...rest}
	onclick={copyText}
>
	<Icon
		src={isCopied ? CheckIcon : CopyIcon}
		class={{
			'text-green-600': isCopied,
			'!size-4': size == 'sm',
			'!size-[14px]': size == 'xs'
		}}
	/>
	{@render children?.()}
</Button>
