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

	type ButtonSize = 'xs' | 'sm' | 'md' | 'lg' | 'mini';

	interface Props extends Omit<ButtonProps, 'size'> {
		icon?: IconComponent;
		size?: ButtonSize;
		tooltip?: string;
		tooltipDelayDuration?: number;
	}

	let {
		icon = X,
		size = 'md',
		tooltip,
		children,
		tooltipDelayDuration,
		...rest
	}: Props = $props();

	//

	type ButtonConfig = {
		iconSize: number;
		sizeClass: string;
	};

	const configs: Record<ButtonSize, ButtonConfig> = {
		xs: {
			iconSize: 14,
			sizeClass: 'size-6'
		},
		sm: {
			iconSize: 16,
			sizeClass: 'size-8'
		},
		md: {
			iconSize: 16,
			sizeClass: 'size-9'
		},
		lg: {
			iconSize: 18,
			sizeClass: 'size-12'
		},
		mini: {
			iconSize: 14,
			sizeClass: 'size-5'
		}
	};

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
		variant="outline"
		{...rest}
		size="icon"
		class={[
			'relative shrink-0',
			{ 'rounded-xs': size === 'mini' },
			currentConfig.sizeClass,
			rest.class
		]}
	>
		<Icon src={icon ?? X} size={currentConfig.iconSize} />
		{@render children?.()}
	</Button>
{/snippet}
