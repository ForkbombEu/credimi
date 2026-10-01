<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { PipelineExecutionArtifacts } from '$lib/pipeline/execution-artifacts';
	import type { Snippet } from 'svelte';

	import { BadgeCheckIcon, FileCogIcon, FileIcon, ImageIcon, VideoIcon } from '@lucide/svelte';
	import { FCAF } from '$lib';
	import MediaPreview from '$lib/components/media-preview.svelte';
	import { mergeProps } from 'bits-ui';

	import type { IconComponent } from '@/components/types';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { m } from '@/i18n';

	import PipelineReportSheet from './pipeline-report-sheet.svelte';

	type PreviewIcon = 'image' | 'video' | 'file' | 'document' | 'fcaf';

	type ArtifactButtonArgs = {
		tooltip: string;
		previewIcon: PreviewIcon;
		icon: IconComponent;
		href?: string;
		image?: string;
		target?: string;
		extraProps?: Record<string, unknown>;
	};

	/** `preview` = media thumbnails; `icons` / `icons-sm` = lucide icon buttons. */
	type Presentation = 'preview' | 'icons' | 'icons-sm';

	type Props = {
		artifacts: PipelineExecutionArtifacts;
		presentation?: Presentation;
		previewClass?: string;
		hideLogs?: boolean;
		emptyState?: Snippet;
	};

	let {
		artifacts,
		presentation = 'preview',
		previewClass,
		hideLogs = false,
		emptyState
	}: Props = $props();

	const ICON_BUTTON_CLASS = 'text-primary hover:bg-secondary';

	const presentationConfig = {
		preview: {
			isPreview: true as const,
			iconSize: 'mini' as const,
			containerClass: 'flex items-center gap-2',
			groupClass: 'flex items-center gap-1',
			iconClass: ICON_BUTTON_CLASS
		},
		icons: {
			isPreview: false as const,
			iconSize: 'xs' as const,
			containerClass: 'flex flex-wrap items-center gap-0',
			groupClass: 'flex items-center gap-0',
			iconClass: `rounded-sm ${ICON_BUTTON_CLASS}`
		},
		'icons-sm': {
			isPreview: false as const,
			iconSize: 'mini' as const,
			containerClass: 'flex flex-wrap items-center gap-1',
			groupClass: 'flex items-center gap-0',
			iconClass: ICON_BUTTON_CLASS
		}
	};

	const config = $derived(presentationConfig[presentation]);

	const hasContent = $derived(
		artifacts.results.length > 0 ||
			(artifacts.maestro_screenshots?.length ?? 0) > 0 ||
			Boolean(artifacts.report) ||
			Boolean(artifacts.fcafReport) ||
			Boolean(artifacts.fcafReportPdf)
	);

	function resultButtons(
		result: PipelineExecutionArtifacts['results'][number]
	): ArtifactButtonArgs[] {
		const buttons: ArtifactButtonArgs[] = [
			{
				tooltip: m.pipeline_artifact_video_tooltip(),
				previewIcon: 'video',
				icon: VideoIcon,
				image: result.screenshot,
				href: result.video,
				target: '_blank'
			},
			{
				tooltip: m.pipeline_artifact_screenshot_tooltip(),
				previewIcon: 'image',
				icon: ImageIcon,
				image: result.screenshot,
				href: result.screenshot,
				target: '_blank'
			}
		];
		if (!hideLogs) {
			buttons.push({
				tooltip: m.pipeline_artifact_log_tooltip(),
				previewIcon: 'file',
				icon: FileCogIcon,
				href: result.log,
				target: '_blank'
			});
		}
		return buttons;
	}
</script>

{#if hasContent}
	<div class={config.containerClass}>
		{#each artifacts.results as result, index (index)}
			<div class={config.groupClass}>
				{#each resultButtons(result) as button (button.previewIcon)}
					{@render artifactButton(button)}
				{/each}
			</div>
		{/each}
		<PipelineReportSheet reportUrl={artifacts.report}>
			{#snippet sheetTrigger({ props })}
				{@render artifactButton({
					tooltip: m.pipeline_artifact_report_tooltip(),
					previewIcon: 'document',
					icon: FileIcon,
					extraProps: props
				})}
			{/snippet}
		</PipelineReportSheet>
		<FCAF.ReportSheet reportUrl={artifacts.fcafReport} pdfUrl={artifacts.fcafReportPdf}>
			{#snippet sheetTrigger({ props })}
				{@render artifactButton({
					tooltip: 'FCAF assessment report',
					previewIcon: 'fcaf',
					icon: BadgeCheckIcon,
					extraProps: props
				})}
			{/snippet}
		</FCAF.ReportSheet>
	</div>
{:else if emptyState}
	{@render emptyState()}
{/if}

{#snippet artifactButton({
	tooltip,
	previewIcon,
	icon,
	href,
	image,
	target,
	extraProps = {}
}: ArtifactButtonArgs)}
	{#if config.isPreview}
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
			size={config.iconSize}
			variant="ghost"
			{icon}
			{href}
			{target}
			class={config.iconClass}
			{tooltip}
			{...extraProps}
		/>
	{/if}
{/snippet}
