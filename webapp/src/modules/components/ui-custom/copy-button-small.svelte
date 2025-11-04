<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { CheckIcon, CopyIcon } from 'lucide-svelte';

	import Button, {
		type ButtonProps,
		type ButtonVariant
	} from '@/components/ui/button/button.svelte';

	import Icon from './icon.svelte';

	//

	type Props = Omit<ButtonProps, 'size'> & {
		delay?: number;
		textToCopy: string;
		children?: Snippet;
		square?: boolean;
		variant?: ButtonVariant;
		size?: ButtonProps['size'] | 'mini';
	};

	let {
		textToCopy,
		delay = 1000,
		children,
		square = false,
		variant = 'outline',
		size = 'default',
		class: className,
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
	class={[
		className,
		{
			'h-8': size === 'default',
			'h-5': size === 'mini',
			'p-0': square,
			'w-8': square && size === 'default',
			'w-5': square && size === 'mini'
		}
	]}
	{...rest}
	onclick={copyText}
>
	<Icon
		src={isCopied ? CheckIcon : CopyIcon}
		class={{
			'text-green-600': isCopied,
			'!size-3': size === 'mini',
			'!size-6': size === 'default'
		}}
	/>
	{@render children?.()}
</Button>
