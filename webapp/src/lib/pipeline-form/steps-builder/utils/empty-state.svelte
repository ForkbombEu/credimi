<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import T from '@/components/ui-custom/t.svelte';

	type Props = {
		text: string;
	};

	let { text }: Props = $props();
</script>

<div class="p-4">
	<div class="flex flex-col items-center justify-center rounded-md bg-slate-100 p-4">
		<T class="text-muted-foreground text-sm">{text}</T>
	</div>
</div>
