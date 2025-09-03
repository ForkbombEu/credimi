<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SetFieldType } from 'type-fest';

	import { Record } from 'effect';
	import { X } from 'lucide-svelte';

	import type { IconComponent } from '../types';
	import type { ButtonProps } from '../ui/button';

	import Button from '../ui/button/button.svelte';
	import Icon from './icon.svelte';

	//

	type ButtonSize = 'md' | 'sm' | 'lg';

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

<Button variant="outline" {...rest} size="icon" class="{currentConfig.sizeClass} {rest.class}">
	<Icon src={icon ?? X} size={currentConfig.iconSize} />
</Button>
