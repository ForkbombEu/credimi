<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ClassValue } from 'svelte/elements';

	import DOMPurify from 'dompurify';

	type Props = {
		content: string;
		class?: ClassValue;
	};

	const { content, class: className }: Props = $props();
</script>

<div class={['prose-strong:text-current prose-a:underline prose-a:underline-offset-1', className]}>
	<!-- eslint-disable-next-line svelte/no-at-html-tags -->
	{@html DOMPurify.sanitize(content)}
</div>
