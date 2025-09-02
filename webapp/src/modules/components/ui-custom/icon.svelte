<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { IconProps } from 'lucide-svelte';

	import type { IconComponent } from '@/components/types';

	import { cn } from '@/components/ui/utils';

	type Props = IconProps & { src: IconComponent; ml?: boolean; mr?: boolean };

	const { src: Src, size = 16, mr, ml, ...rest }: Props = $props();

	const classes = $derived(
		cn(rest.class, rest.className, {
			'mr-2': mr,
			'ml-2': ml
		})
	);
</script>

<Src {size} {...rest} class={classes} />
