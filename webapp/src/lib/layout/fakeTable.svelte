<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Ellipsis } from 'lucide-svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import Input from '@/components/ui/input/input.svelte';
</script>

<div class="flex gap-4 blur-sm">
	<Button variant="outline" disabled>Sort</Button>
	<Input disabled class="hover:cursor-pointer" />
	<Button variant="outline" disabled>View types</Button>
</div>
<table class="w-full rounded-xl border bg-white blur-sm">
	<thead class="bg-slate-100">
		<tr>
			<th>Status</th>
			<th>Test</th>
			<th>Last updates</th>
			<th></th>
		</tr>
	</thead>
	<tbody>
		<tr>
			<td class="text-green-700">Passed</td>
			<td>Sd-jwt</td>
			<td>2024-12-42</td>
			<td><Ellipsis /></td>
		</tr>
		<tr>
			<td class="text-red-700">Failed</td>
			<td>Sd-jwt</td>
			<td>2024-12-42</td>
			<td><Ellipsis /></td>
		</tr>
		<tr>
			<td class="text-green-700">Passed</td>
			<td>Sd-jwt</td>
			<td>2024-12-42</td>
			<td><Ellipsis /></td>
		</tr>
		<tr>
			<td class="text-green-700">Passed</td>
			<td>Sd-jwt</td>
			<td>2024-12-42</td>
			<td><Ellipsis /></td>
		</tr>
		<tr>
			<td class="text-red-700">Passed</td>
			<td>Sd-jwt</td>
			<td>2024-12-42</td>
			<td><Ellipsis /></td>
		</tr>
	</tbody>
</table>

<style lang="postcss">
	tr {
		@apply border-y;
	}

	td,
	th {
		@apply p-4;
	}

	th {
		@apply text-left;
	}
</style>
