<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import MediaPreview from '$lib/components/media-preview.svelte';
	import { nanoid } from 'nanoid';

	import type { PipelineResultsResponse } from '@/pocketbase/types';

	import { pb } from '@/pocketbase';

	import * as Column from '../column';
	import Na from './partials/na.svelte';

	//

	export const column = Column.define({
		fn: (row) => {
			const results = row.expand.latest_successful_execution;
			if (!results) return [];
			else return groupExecutionArtifacts(results, row.mobile_runners);
		},
		id: 'video_screenshot',
		header: ''
	});

	function groupExecutionArtifacts(res: PipelineResultsResponse, runners: string[]) {
		return runners.map((runner) => {
			const screenshot = res.screenshots.find((s) => s.startsWith(runner));
			const screenshotUrl = screenshot ? pb.files.getURL(res, screenshot) : undefined;
			const video = res.video_results.find((v) => v.startsWith(runner));
			const videoUrl = video ? pb.files.getURL(res, video) : undefined;
			return {
				id: nanoid(3),
				screenshot: screenshotUrl,
				video: videoUrl
			};
		});
	}
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

<div class="flex items-center gap-2">
	{#each value as item (item.id)}
		<div class="flex items-center gap-1">
			{#if item.screenshot}
				<MediaPreview image={item.screenshot} href={item.screenshot} icon="image" />
			{/if}
			{#if item.video}
				<MediaPreview image={item.screenshot} href={item.video} icon="video" />
			{/if}
		</div>
	{:else}
		<Na />
	{/each}
</div>
