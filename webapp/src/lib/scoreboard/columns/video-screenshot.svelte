<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { fromEnrichedRecord } from '$lib/pipeline/execution-artifacts';
	import ExecutionArtifactsPreview from '$lib/pipeline/results/execution-artifacts-preview.svelte';

	import * as Column from '../column';
	import * as EntityDisplay from '../entity-display';

	//

	export const column = Column.define({
		fn: (row) =>
			fromEnrichedRecord(
				(row.expand.latest_successful_execution ?? {}) as Parameters<typeof fromEnrichedRecord>[0]
			),
		id: 'video_screenshot',
		header: ' '
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

{#if value}
	<ExecutionArtifactsPreview artifacts={value} variant="preview" previewClass="size-8!" />
{:else}
	<EntityDisplay.Na />
{/if}
