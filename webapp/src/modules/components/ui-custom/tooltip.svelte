<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import * as Tooltip from '@/components/ui/tooltip/index.js';

	type Props = {
		children?: Snippet;
		content?: Snippet;
		delayDuration?: number;
	};

	let { children, content, delayDuration = 500 }: Props = $props();
</script>

<Tooltip.Provider {delayDuration}>
	<Tooltip.Root>
		<Tooltip.Trigger>{@render children?.()}</Tooltip.Trigger>
		<Tooltip.Content>
			{@render content?.()}
		</Tooltip.Content>
	</Tooltip.Root>
</Tooltip.Provider>
