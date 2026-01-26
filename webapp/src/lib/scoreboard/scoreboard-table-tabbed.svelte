<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV
SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { m } from '@/i18n';

	import type { ScoreboardData } from './types';

	import ScoreboardTable from './scoreboard-table.svelte';

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
		{ key: 'wallets' as const, label: m.wallets(), count: data.summary.wallets.length },
		{ key: 'issuers' as const, label: m.issuers(), count: data.summary.issuers.length },
		{ key: 'verifiers' as const, label: m.verifiers(), count: data.summary.verifiers.length },
		{ key: 'pipelines' as const, label: m.pipelines(), count: data.summary.pipelines.length }
	];
</script>

<!-- Tab Navigation -->
<div class="tab-navigation">
	{#each tabs as tab (tab.key)}
		<button
			class="tab-button"
			class:tab-active={activeTab === tab.key}
			class:tab-inactive={activeTab !== tab.key}
			onclick={() => (activeTab = tab.key)}
		>
			{tab.label} ({tab.count})
		</button>
	{/each}
</div>

<!-- Data Table -->
<ScoreboardTable
	entries={getTabData()}
	{showActions}
	emptyMessage={m.scoreboard_no_data_for_tab({ tab: activeTab })}
/>

<style lang="postcss">
	@reference "tailwindcss";

	.tab-navigation {
		@apply mb-6 flex gap-2 border-b;
	}

	.tab-button {
		@apply px-4 py-2;
	}

	.tab-active {
		@apply border-b-2 border-blue-600 font-semibold text-blue-600;
	}

	.tab-inactive {
		@apply text-slate-600 hover:text-blue-600;
	}
</style>
