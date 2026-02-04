<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ComponentProps } from 'svelte';

	import type { IconComponent } from '@/components/types';

	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';

	//

	type Props = Omit<ComponentProps<typeof Button>, 'size'> & {
		icon?: IconComponent;
		compact?: boolean;
	};

	let {
		icon,
		compact = false,
		variant = 'link',
		class: className,
		children,
		...rest
	}: Props = $props();
</script>

{#if compact}
	<div class="flex h-5! items-center justify-center">
		{@render content()}
	</div>
{:else}
	{@render content()}
{/if}

{#snippet content()}
	<Button
		size="sm"
		{variant}
		class={[
			'h-8 gap-1 px-2 text-blue-600 hover:cursor-pointer hover:bg-blue-50 hover:no-underline',
			className
		]}
		{...rest}
	>
		{#if icon}
			<Icon src={icon} />
		{/if}
		{@render children?.()}
	</Button>
{/snippet}
