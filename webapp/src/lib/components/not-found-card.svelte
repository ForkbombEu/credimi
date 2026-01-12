<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { TriangleAlert } from 'lucide-svelte';

	import { m } from '@/i18n';

	type Props = {
		class?: string;
	};

	let { class: className = '' }: Props = $props();
</script>

<div
	class={[
		'bg-card border-destructive flex flex-col gap-3 rounded-lg border p-4 shadow-sm',
		className
	]}
>
	<div class="flex items-center gap-2">
		<TriangleAlert class="text-destructive h-5 w-5" />
		<h3 class="font-semibold">{m.enrich_error_title()}</h3>
	</div>
	<p class="text-muted-foreground text-sm">
		{m.enrich_error_description()}
	</p>
</div>
