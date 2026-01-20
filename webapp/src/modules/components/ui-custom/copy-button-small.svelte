<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CheckIcon, CopyIcon } from '@lucide/svelte';

	import Button, { type ButtonProps } from '@/components/ui/button/button.svelte';
	import * as Tooltip from '@/components/ui/tooltip';
	import { m } from '@/i18n';

	import Icon from './icon.svelte';

	//

	type Props = Omit<ButtonProps, 'size'> & {
		delay?: number;
		textToCopy: string;
		size?: ButtonProps['size'] | 'xs' | 'mini';
		hideTooltip?: boolean;
	};

	let {
		textToCopy,
		delay = 1000,
		children,
		variant = 'ghost',
		size = 'xs',
		class: className,
		hideTooltip = false,
		...rest
	}: Props = $props();

	//

	const square = $derived(!children);

	let isCopied = $state(false);

	function copyText() {
		navigator.clipboard.writeText(textToCopy);
		isCopied = true;
		setTimeout(() => {
			isCopied = false;
		}, delay);
	}
</script>

{#if !hideTooltip}
	<Tooltip.Provider>
		<Tooltip.Root>
			<Tooltip.Trigger>
				{#snippet child({ props })}
					{@render button(props)}
				{/snippet}
			</Tooltip.Trigger>
			<Tooltip.Content class="text-xs! max-w-[600px] overflow-auto">
				<p>{m.Copy()}: <span class="font-mono">{textToCopy}</span></p>
			</Tooltip.Content>
		</Tooltip.Root>
	</Tooltip.Provider>
{:else}
	{@render button()}
{/if}

{#snippet button(props?: Record<string, unknown>)}
	<Button
		{variant}
		class={[
			'text-gray-400',
			{
				'h-8': size == 'sm',
				'h-6': size == 'xs',
				'h-5': size === 'mini',
				'flex items-center justify-center p-0!': square,
				'w-8': square && (size == 'sm' || size === 'default'),
				'w-6': square && size == 'xs',
				'w-5': square && size === 'mini',
				'rounded-sm! text-xs': size === 'mini',
				"px-1": size === 'mini' && !square,
			},
			className
		]}
		{...rest}
		{...props}
		onclick={(e) => {
			e.stopPropagation();
			copyText();
		}}
	>
		<Icon
			src={isCopied ? CheckIcon : CopyIcon}
			class={{
				'text-green-600': isCopied,
				'size-4!': size == 'sm',
				'size-[14px]!': size == 'xs',
				'size-3!': size === 'mini',
				'size-6!': size === 'default'
			}}
		/>
		{@render children?.()}
	</Button>
{/snippet}
