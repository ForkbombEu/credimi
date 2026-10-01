<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { X } from '@lucide/svelte';
	import { mergeProps } from 'bits-ui';

	import type { IconComponent } from '../types';
	import type { ButtonProps } from '../ui/button';

	import Button from './button.svelte';
	import Icon from './icon.svelte';
	import Tooltip from './tooltip.svelte';

	//

	type IconButtonSize = 'xs' | 'sm' | 'md' | 'lg' | 'mini';

	interface Props extends Omit<ButtonProps, 'size'> {
		icon?: IconComponent;
		size?: IconButtonSize;
		tooltip?: string;
		tooltipDelayDuration?: number;
	}

	let {
		icon = X,
		size = 'md',
		variant = 'outline',
		tooltip,
		children,
		tooltipDelayDuration,
		class: className,
		...rest
	}: Props = $props();

	//

	type ButtonConfig = {
		iconSize: number;
		sizeClass: string;
	};

	const configs: Record<IconButtonSize, ButtonConfig> = {
		lg: {
			sizeClass: '!size-12',
			iconSize: 18
		},
		md: {
			sizeClass: '!size-9',
			iconSize: 16
		},
		sm: {
			sizeClass: '!size-8',
			iconSize: 16
		},
		xs: {
			sizeClass: '!size-6',
			iconSize: 15
		},
		mini: {
			sizeClass: '!size-5',
			iconSize: 14
		}
	};

	const primitiveSizeByIconSize = {
		mini: 'icon-sm',
		xs: 'icon-sm',
		sm: 'icon-sm',
		md: 'icon',
		lg: 'icon-lg'
	} as const satisfies Record<IconButtonSize, NonNullable<ButtonProps['size']>>;

	const currentConfig = $derived(configs[size]);
</script>

{#if tooltip}
	<Tooltip delayDuration={tooltipDelayDuration}>
		{#snippet child({ props })}
			{@render button(props)}
		{/snippet}

		{#snippet content()}
			<p>{tooltip}</p>
		{/snippet}
	</Tooltip>
{:else}
	{@render button({})}
{/if}

{#snippet button(props: Record<string, unknown>)}
	{@const mergedProps = mergeProps(rest, props)}
	<Button
		{...mergedProps}
		{variant}
		size={primitiveSizeByIconSize[size]}
		class={[
			'relative shrink-0',
			{ 'rounded-xs': size === 'mini' },
			currentConfig.sizeClass,
			className
		]}
	>
		<Icon
			src={icon ?? X}
			size={currentConfig.iconSize}
			class="size-[var(--icon-button-glyph-size)]"
			style={`--icon-button-glyph-size: ${currentConfig.iconSize}px`}
		/>
		{@render children?.()}
	</Button>
{/snippet}
