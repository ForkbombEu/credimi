<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { X } from 'lucide-svelte';

	import type { IconComponent } from '../types';
	import type { ButtonProps } from '../ui/button';

	import Button from './button.svelte';
	import Icon from './icon.svelte';

	//

	type ButtonSize = 'xs' | 'sm' | 'md' | 'lg';

	interface Props extends Omit<ButtonProps, 'size'> {
		icon?: IconComponent;
		size?: ButtonSize;
	}

	let { icon = X, size = 'md', ...rest }: Props = $props();

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
			sizeClass: 'size-10'
		},
		lg: {
			iconSize: 18,
			sizeClass: 'size-12'
		}
	};

	const currentConfig = $derived(configs[size]);
</script>

<Button
	variant="outline"
	{...rest}
	size="icon"
	class={['shrink-0', currentConfig.sizeClass, rest.class]}
>
	<Icon src={icon ?? X} size={currentConfig.iconSize} />
</Button>
