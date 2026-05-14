<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { featureFlags } from '@/features';
</script>

<div class="space-y-8 p-4">
	<h1>Features data</h1>
	{#each Object.entries($featureFlags) as [name, flag] (name)}
		<div class="space-y-2">
			<p>Feature: {name} | Active: {flag}</p>
			<div class="flex items-stretch gap-4">
				<div class="w-0 grow overflow-x-auto rounded-lg bg-gray-200 p-4 font-mono">
					{#if flag}
						<div>
							This is feature {name} and it is {flag ? 'active' : 'not active'}
						</div>
					{:else}
						<p class="text-gray-400">This feature is not active</p>
					{/if}
				</div>
			</div>
		</div>
	{/each}
</div>
