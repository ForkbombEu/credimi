<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV
SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import ScoreboardTable from './scoreboard-table.svelte';
	import type { ScoreboardData } from './types';

	interface Props {
		data: ScoreboardData;
		showActions?: boolean;
	}

	let { data, showActions = false }: Props = $props();

	let activeTab: 'wallets' | 'issuers' | 'verifiers' | 'pipelines' = $state('wallets');

	function getTabData() {
		return data.summary[activeTab] || [];
	}

	const tabs = [
		{ key: 'wallets' as const, label: 'Wallets', count: data.summary.wallets.length },
		{ key: 'issuers' as const, label: 'Issuers', count: data.summary.issuers.length },
		{ key: 'verifiers' as const, label: 'Verifiers', count: data.summary.verifiers.length },
		{ key: 'pipelines' as const, label: 'Pipelines', count: data.summary.pipelines.length }
	];
</script>

<!-- Tab Navigation -->
<div class="mb-6 flex gap-2 border-b">
	{#each tabs as tab}
		<button
			class="px-4 py-2 {activeTab === tab.key
				? 'border-b-2 border-blue-600 font-semibold text-blue-600'
				: 'text-gray-600 hover:text-blue-600'}"
			onclick={() => (activeTab = tab.key)}
		>
			{tab.label} ({tab.count})
		</button>
	{/each}
</div>

<!-- Data Table -->
<ScoreboardTable entries={getTabData()} {showActions} emptyMessage="No {activeTab} data available" />
