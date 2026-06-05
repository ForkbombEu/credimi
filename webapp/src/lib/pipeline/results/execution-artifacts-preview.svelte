<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { PipelineExecutionArtifacts } from '$lib/pipeline/execution-artifacts';
	import type { Snippet } from 'svelte';

	import { FileCogIcon, FileIcon, ImageIcon, VideoIcon } from '@lucide/svelte';
	import MediaPreview from '$lib/components/media-preview.svelte';
	import { mergeProps } from 'bits-ui';

	import type { IconComponent } from '@/components/types';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { m } from '@/i18n';

	import PipelineReportSheet from './pipeline-report-sheet.svelte';

	type PreviewIcon = 'image' | 'video' | 'file' | 'document';

	type ArtifactButtonArgs = {
		tooltip: string;
		previewIcon: PreviewIcon;
		compactIcon: IconComponent;
		href?: string;
		image?: string;
		target?: string;
		extraProps?: Record<string, unknown>;
	};

	type Props = {
		artifacts: PipelineExecutionArtifacts;
		variant?: 'preview' | 'compact';
		previewClass?: string;
		hideLogs?: boolean;
		emptyState?: Snippet;
	};

	let {
		artifacts,
		variant = 'preview',
		previewClass,
		hideLogs = false,
		emptyState
	}: Props = $props();

	const hasContent = $derived(artifacts.results.length > 0 || Boolean(artifacts.report));

	const containerClass = $derived(
		variant === 'compact' ? 'flex flex-wrap items-center gap-1' : 'flex items-center gap-2'
	);
</script>

{#if hasContent}
	<div class={containerClass}>
		{#each artifacts.results as result, index (index)}
			<div class="flex items-center gap-1">
				{@render artifactButton({
					tooltip: m.pipeline_artifact_video_tooltip(),
					previewIcon: 'video',
					compactIcon: VideoIcon,
					image: result.screenshot,
					href: result.video,
					target: '_blank'
				})}
				{@render artifactButton({
					tooltip: m.pipeline_artifact_screenshot_tooltip(),
					previewIcon: 'image',
					compactIcon: ImageIcon,
					image: result.screenshot,
					href: result.screenshot,
					target: '_blank'
				})}
				{#if !hideLogs}
					{@render artifactButton({
						tooltip: m.pipeline_artifact_log_tooltip(),
						previewIcon: 'file',
						compactIcon: FileCogIcon,
						href: result.log,
						target: '_blank'
					})}
				{/if}
			</div>
		{/each}
		<PipelineReportSheet reportUrl={artifacts.report}>
			{#snippet sheetTrigger({ props })}
				{@render artifactButton({
					tooltip: m.pipeline_artifact_report_tooltip(),
					previewIcon: 'document',
					compactIcon: FileIcon,
					extraProps: props
				})}
			{/snippet}
		</PipelineReportSheet>
	</div>
{:else if emptyState}
	{@render emptyState()}
{/if}

{#snippet artifactButton({
	tooltip,
	previewIcon,
	compactIcon,
	href,
	image,
	target,
	extraProps = {}
}: ArtifactButtonArgs)}
	{#if variant === 'preview'}
		<Tooltip>
			{#snippet child({ props })}
				<MediaPreview
					{image}
					{href}
					icon={previewIcon}
					class={previewClass}
					{...mergeProps(extraProps, props)}
				/>
			{/snippet}

			{#snippet content()}
				<p>{tooltip}</p>
			{/snippet}
		</Tooltip>
	{:else}
		<IconButton
			size="mini"
			variant="ghost"
			icon={compactIcon}
			{href}
			{target}
			class="text-primary hover:bg-secondary"
			{tooltip}
			{...extraProps}
		/>
	{/if}
{/snippet}
