<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ComponentProps } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	import type { WalletActionsResponse } from '@/pocketbase/types';

	import { Badge } from '@/components/ui/badge';

	//

	type Props = ComponentProps<typeof Badge> & {
		action: WalletActionsResponse;
		containerClass?: ClassValue;
	};

	let { action, containerClass, ...rest }: Props = $props();

	const tags = $derived(action.tags.split(',').map((tag) => tag.trim()));
</script>

<div class={['flex items-center gap-1 pr-2', containerClass]}>
	{#each tags as tag, index (index)}
		<Badge variant="secondary" {...rest}>
			{tag}
		</Badge>
	{/each}
</div>
