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
				(row.expanded_data?.latest_execution ?? {}) as Parameters<
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
	import { buildPipelineResultViewerHref } from '$lib/pipeline/results/maestro-timeline';

	import IconButton from '@/components/ui-custom/iconButton.svelte';

	let { value }: Column.Props<typeof column> = $props();

	const result = $derived(value?.results[0]);
	const videoHref = $derived(result && buildPipelineResultViewerHref('video', result));
	const screenshotHref = $derived(result && buildPipelineResultViewerHref('screenshot', result));
</script>

{#if value && (videoHref || screenshotHref || value.fcafReportPdf)}
	<div class="flex items-center gap-1">
		{#if videoHref}
			<IconButton
				size="mini"
				variant="ghost"
				icon={VideoIcon}
				href={videoHref}
				target="_blank"
				class="text-primary hover:bg-secondary"
				tooltip={m.pipeline_artifact_video_tooltip()}
			/>
		{/if}
		{#if screenshotHref}
			<IconButton
				size="mini"
				variant="ghost"
				icon={ImageIcon}
				href={screenshotHref}
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
