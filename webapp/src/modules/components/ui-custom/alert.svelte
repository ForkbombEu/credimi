<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { tv, type VariantProps } from 'tailwind-variants';

	import { alertVariants } from '@/components/ui/alert/index.js';

	const variants = tv({
		extend: alertVariants,
		variants: {
			variant: {
				info: 'border-blue-600/50 text-blue-600 dark:border-blue-600 [&>svg]:text-blue-600',
				success:
					'border-green-600/50 text-green-600 dark:border-green-600 [&>svg]:text-green-600',
				warning:
					'border-yellow-600/50 text-yellow-600 dark:border-yellow-600 [&>svg]:text-yellow-600'
			}
		}
	});

	type AlertVariant = VariantProps<typeof variants>['variant'];
</script>

<script lang="ts">
	import type { ComponentProps, Snippet } from 'svelte';

	import type { IconComponent } from '@/components/types';

	import * as Alert from '@/components/ui/alert/index.js';

	import Icon from './icon.svelte';

	type Props = Omit<ComponentProps<typeof Alert.Root>, 'variant'> & {
		content?: Snippet<[{ Title: typeof Alert.Title; Description: typeof Alert.Description }]>;
		children?: Snippet;
		icon?: IconComponent;
		variant?: AlertVariant;
	};

	const { children, content, icon, variant, class: className, ...alertProps }: Props = $props();
</script>

<Alert.Root {...alertProps} class={[variants({ variant }), className, { 'p-4!': !icon }]}>
	{#if icon}
		<Icon src={icon} size={16} />
	{/if}

	{@render content?.({ Title: Alert.Title, Description: Alert.Description })}

	{@render children?.()}
</Alert.Root>
