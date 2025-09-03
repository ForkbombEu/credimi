<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import DOMPurify from 'dompurify';

	import { cn } from '../ui/utils';

	type Props = {
		content: string;
		class?: string;
	};

	const { content, class: className }: Props = $props();
</script>

<div class={cn('prose-strong:text-current', className)}>
	{@html DOMPurify.sanitize(content)}
</div>
