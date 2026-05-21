<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { X } from '@lucide/svelte';

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
		xs: {
			iconSize: 14,
			sizeClass: '!size-6'
		},
		sm: {
			iconSize: 16,
			sizeClass: '!size-8'
		},
		md: {
			iconSize: 16,
			sizeClass: '!size-9'
		},
		lg: {
			iconSize: 18,
			sizeClass: '!size-12'
		},
		mini: {
			iconSize: 14,
			sizeClass: '!size-5'
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
		{@render button()}
		{#snippet content()}
			<p>{tooltip}</p>
		{/snippet}
	</Tooltip>
{:else}
	{@render button()}
{/if}

{#snippet button()}
	<Button
		{...rest}
		{variant}
		size={primitiveSizeByIconSize[size]}
		class={[
			'relative shrink-0',
			{ 'rounded-xs': size === 'mini' },
			currentConfig.sizeClass,
			className
		]}
	>
		<Icon src={icon ?? X} size={currentConfig.iconSize} />
		{@render children?.()}
	</Button>
{/snippet}
