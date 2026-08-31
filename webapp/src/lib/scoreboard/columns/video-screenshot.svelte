<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { fromEnrichedRecord } from '$lib/pipeline/execution-artifacts';

	import { renderComponent } from '@/components/ui/data-table';
	import { m } from '@/i18n';

	import * as Column from '../column';
	import * as EntityDisplay from '../entity-display';
	import EntityHeader from './headers/entity-header.svelte';

	//

	export const column = Column.define({
		fn: (row) =>
			fromEnrichedRecord(
				(row.expand.latest_successful_execution ?? {}) as Parameters<
					typeof fromEnrichedRecord
				>[0]
			),
		id: 'evidence',
		header: renderComponent(EntityHeader, {
			label: m.Evidence()
		})
	});
</script>

<script lang="ts">
	import { BadgeCheckIcon, ImageIcon, VideoIcon } from '@lucide/svelte';

	import IconButton from '@/components/ui-custom/iconButton.svelte';

	let { value }: Column.Props<typeof column> = $props();

	const result = $derived(value?.results[0]);
</script>

{#if value && (result || value.fcafReportPdf)}
	<div class="flex items-center gap-1">
		{#if result}
			<IconButton
				size="mini"
				variant="ghost"
				icon={VideoIcon}
				href={result.video}
				target="_blank"
				class="text-primary hover:bg-secondary"
				tooltip={m.pipeline_artifact_video_tooltip()}
			/>
			<IconButton
				size="mini"
				variant="ghost"
				icon={ImageIcon}
				href={result.screenshot}
				target="_blank"
				class="text-primary hover:bg-secondary"
				tooltip={m.pipeline_artifact_screenshot_tooltip()}
			/>
		{/if}
		{#if value.fcafReportPdf}
			<IconButton
				size="mini"
				variant="ghost"
				icon={BadgeCheckIcon}
				href={value.fcafReportPdf}
				target="_blank"
				class="text-primary hover:bg-secondary"
				tooltip="FCAF PDF report"
			/>
		{/if}
	</div>
{:else}
	<EntityDisplay.Na />
{/if}
