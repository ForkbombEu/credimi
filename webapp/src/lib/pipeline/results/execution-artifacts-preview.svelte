<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';
	import { FileCogIcon, FileIcon, ImageIcon, VideoIcon } from '@lucide/svelte';

	import MediaPreview from '$lib/components/media-preview.svelte';
	import type { PipelineExecutionArtifacts } from '$lib/pipeline/execution-artifacts';

	import IconButton from '@/components/ui-custom/iconButton.svelte';

	import PipelineReportSheet from './pipeline-report-sheet.svelte';

	type Props = {
		artifacts: PipelineExecutionArtifacts;
		variant?: 'preview' | 'compact';
		previewClass?: string;
		emptyState?: Snippet;
	};

	let { artifacts, variant = 'preview', previewClass, emptyState }: Props = $props();

	const hasContent = $derived(artifacts.results.length > 0 || Boolean(artifacts.report));

	const containerClass = $derived(
		variant === 'compact' ? 'flex flex-wrap items-center gap-1' : 'flex items-center gap-2'
	);
</script>

{#if hasContent}
	<div class={containerClass}>
		{#each artifacts.results as result, index (index)}
			<div class="flex items-center gap-1">
				{#if variant === 'preview'}
					<MediaPreview
						image={result.screenshot}
						href={result.video}
						icon="video"
						class={previewClass}
					/>
					<MediaPreview
						image={result.screenshot}
						href={result.screenshot}
						icon="image"
						class={previewClass}
					/>
					<MediaPreview href={result.log} icon="file" class={previewClass} />
				{:else}
					<IconButton
						size="mini"
						variant="ghost"
						icon={VideoIcon}
						href={result.video}
						target="_blank"
						class="text-primary hover:bg-secondary"
					/>
					<IconButton
						size="mini"
						variant="ghost"
						icon={ImageIcon}
						href={result.screenshot}
						target="_blank"
						class="text-primary hover:bg-secondary"
					/>
					<IconButton
						size="mini"
						variant="ghost"
						icon={FileCogIcon}
						href={result.log}
						target="_blank"
						class="text-primary hover:bg-secondary"
					/>
				{/if}
			</div>
		{/each}
		<PipelineReportSheet reportUrl={artifacts.report}>
			{#snippet sheetTrigger({ props })}
				{#if variant === 'preview'}
					<MediaPreview icon="document" class={previewClass} {...props} />
				{:else}
					<IconButton
						size="mini"
						variant="ghost"
						icon={FileIcon}
						class="text-primary hover:bg-secondary"
						{...props}
					/>
				{/if}
			{/snippet}
		</PipelineReportSheet>
	</div>
{:else if emptyState}
	{@render emptyState()}
{/if}
