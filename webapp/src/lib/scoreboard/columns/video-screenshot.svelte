<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import MediaPreview from '$lib/components/media-preview.svelte';
	import { groupBy } from 'effect/Array';
	import { nanoid } from 'nanoid';

	import type { PipelineResultsResponse } from '@/pocketbase/types';

	import RenderMD from '@/components/ui-custom/renderMD.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import { pb } from '@/pocketbase';

	import * as Column from '../column';
	import * as EntityDisplay from '../entity-display';

	//

	export const column = Column.define({
		fn: (row) => {
			const latestResults = row.expand.latest_successful_execution;
			if (!latestResults) return undefined;
			const groups = groupExecutionArtifacts(latestResults);
			const reportPath = pb.files.getURL(latestResults, latestResults.report);
			return { groups, reportPath };
		},
		id: 'video_screenshot',
		header: ' '
	});

	type ExecutionArtifact = {
		id: string;
		video: string | undefined;
		screenshot: string | undefined;
	};

	function groupExecutionArtifacts(res: PipelineResultsResponse): ExecutionArtifact[] {
		const videoDelimiter = '_result_video_';
		const screenshotDelimiter = '_screenshot_';

		const screenshots = res.screenshots.map((s) => ({
			filename: pb.files.getURL(res, s),
			key: s.split(screenshotDelimiter).at(0) ?? '',
			type: 'screenshot' as const
		}));
		const videos = res.video_results.map((v) => ({
			filename: pb.files.getURL(res, v),
			key: v.split(videoDelimiter).at(0) ?? '',
			type: 'video' as const
		}));

		const groups = groupBy([...screenshots, ...videos], (e) => e.key);
		return Object.values(groups).map((value) => {
			return {
				id: nanoid(3),
				video: value.find((e) => e.type === 'video')?.filename,
				screenshot: value.find((e) => e.type === 'screenshot')?.filename
			};
		});
	}
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();

	const reportPromise = $derived.by(() => {
		if (!value?.reportPath) return undefined;
		return fetch(value.reportPath).then((res) => res.text());
	});
</script>

{#if value && (value.groups.length > 0 || value.reportPath)}
	<div class="flex items-center gap-2 pr-4">
		{#each value.groups as item (item.id)}
			<div class="flex items-center gap-1">
				{#if item.screenshot}
					<MediaPreview
						image={item.screenshot}
						href={item.screenshot}
						icon="image"
						class="size-8!"
					/>
				{/if}
				{#if item.video}
					<MediaPreview
						image={item.screenshot}
						href={item.video}
						icon="video"
						class="size-8!"
					/>
				{/if}
			</div>
		{/each}
		{#await reportPromise then report}
			{#if report}
				<Sheet>
					{#snippet trigger({ sheetTriggerAttributes })}
						<MediaPreview icon="document" class="size-8!" {...sheetTriggerAttributes} />
					{/snippet}
					{#snippet content()}
						<div class="max-w-full min-w-0 p-4">
							<RenderMD
								content={report}
								scrollableTables
								class="prose prose-sm max-w-none prose-headings:text-primary prose-a:text-primary [&_th]:bg-secondary [&_th]:pt-2"
							/>
						</div>
					{/snippet}
				</Sheet>
			{/if}
		{/await}
	</div>
{:else}
	<EntityDisplay.Na />
{/if}
