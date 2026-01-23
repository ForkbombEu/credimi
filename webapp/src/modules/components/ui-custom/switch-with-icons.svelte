<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Switch as SwitchPrimitive } from 'bits-ui';

	import type { IconComponent } from '../types';

	import Icon from './icon.svelte';

	//

	type Size = 'sm' | 'md';

	type Props = {
		offIcon: IconComponent;
		onIcon: IconComponent;
		size?: 'sm' | 'md';
	} & SwitchPrimitive.RootProps;

	let {
		ref = $bindable(null),
		class: className,
		checked = $bindable(false),
		offIcon,
		onIcon,
		size = 'md',
		...restProps
	}: Props = $props();

	//

	const sizes: Record<Size, { container: string; thumb: string; translate: string }> = {
		sm: {
			container: 'h-8 w-14',
			thumb: 'h-7 w-7',
			translate: 'data-[state=checked]:translate-x-6'
		},
		md: {
			container: 'h-9 w-15',
			thumb: 'h-8 w-8',
			translate: 'data-[state=checked]:translate-x-6'
		}
	};
	const currentSize = $derived(sizes[size]);
</script>

<SwitchPrimitive.Root
	bind:ref
	bind:checked
	class={[
		'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
		'data-[state=checked]:bg-green-600 data-[state=unchecked]:bg-input',
		'peer inline-flex shrink-0 items-center rounded-md border-2 border-transparent',
		'cursor-pointer transition-colors',
		'disabled:cursor-not-allowed disabled:opacity-50',
		currentSize.container,
		className
	]}
	{...restProps}
>
	<SwitchPrimitive.Thumb
		class={[
			'pointer-events-none block rounded-sm bg-background shadow-lg ring-0 transition-transform',
			currentSize.translate,
			'flex items-center justify-center',
			currentSize.thumb
		]}
	>
		<Icon src={checked ? onIcon : offIcon} />
	</SwitchPrimitive.Thumb>
</SwitchPrimitive.Root>
