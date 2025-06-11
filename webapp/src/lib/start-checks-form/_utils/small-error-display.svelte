<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { m } from '@/i18n';

	type Props = {
		error: Error;
	};

	let { error }: Props = $props();
</script>

<div class="rounded-md border border-red-500 bg-red-100 px-1 py-0.5 text-xs text-red-700">
	<p class="space-x-1">
		<span class="font-bold">{m.Error()}:</span>
		<span>{error.message}</span>
	</p>
</div>
