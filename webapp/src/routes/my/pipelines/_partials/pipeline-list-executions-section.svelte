<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type {
		PipelineListExecutions,
		PipelineListExecutionsSection
	} from './pipeline-list-executions.svelte.js';

	type Props = {
		store: PipelineListExecutions;
		section: PipelineListExecutionsSection;
		ids: string[];
	};

	let { store, section, ids }: Props = $props();

	$effect(() => {
		store.setSection(section, ids);
	});
</script>
