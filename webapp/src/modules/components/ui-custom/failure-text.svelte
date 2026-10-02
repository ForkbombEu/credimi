<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ClassValue } from 'clsx';
	import type { Snippet } from 'svelte';

	//

	type Props = {
		children?: Snippet;
		class?: ClassValue;
	};

	let { class: className, children }: Props = $props();
</script>

<div
	class={[
		'max-h-96 overflow-auto font-sans text-sm leading-relaxed whitespace-pre-wrap',
		className
	]}
>
	{@render children?.()}
</div>
