<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Attachment } from 'svelte/attachments';

	import { DownloadIcon } from '@lucide/svelte';
	import {
		maestroFlowsAt,
		parseMaestroTimeline,
		type MaestroRowView,
		type MaestroTimeline
	} from '$lib/pipeline/results/maestro-timeline';
	import MaestroTimelineBox from '$lib/pipeline/results/maestro-timeline-box.svelte';
	import { onMount } from 'svelte';

	import { m } from '@/i18n';

	let { data } = $props();

	let timeline = $state<MaestroTimeline | undefined>();
	let loaded = $state(false);
	let aspect = $state<number | undefined>();
	let currentTime = $state(0);
	let videoElement: HTMLVideoElement | undefined;
	let frame = 0;

	onMount(() => {
		let destroyed = false;
		fetch(data.timeline)
			.then((res) => {
				if (!res.ok) throw new Error(`timeline: HTTP ${res.status}`);
				return res.text();
			})
			.then((text) => {
				if (destroyed) return;
				timeline = parseMaestroTimeline(text);
			})
			.catch(() => {
				timeline = undefined;
			})
			.finally(() => {
				loaded = true;
			});
		return () => {
			destroyed = true;
		};
	});

	const startedAt = $derived(data.mode === 'video' ? timeline?.recordingStartedAt : undefined);

	const flows = $derived.by(() => {
		if (!timeline) return [];
		if (startedAt === undefined) return maestroFlowsAt(timeline, Infinity);
		return maestroFlowsAt(timeline, startedAt + currentTime * 1000);
	});

	const trackVideo: Attachment<HTMLVideoElement> = (video) => {
		videoElement = video;
		return () => {
			cancelAnimationFrame(frame);
			videoElement = undefined;
		};
	};

	// While playing, follow the video frame by frame; `timeupdate` is too coarse.
	function followPlayback() {
		if (!videoElement) return;
		currentTime = videoElement.currentTime;
		if (!videoElement.paused && !videoElement.ended)
			frame = requestAnimationFrame(followPlayback);
	}

	function seekTo(row: MaestroRowView) {
		if (!videoElement || startedAt === undefined) return;
		videoElement.currentTime = Math.max(0, (row.startedAt - startedAt) / 1000);
	}

	// PocketBase serves `?download=1` as an attachment; `download` alone covers browsers
	// that honour it, since result URLs are always on this page's origin.
	const videoDownload = $derived.by(() => {
		if (!data.video) return undefined;
		const url = new URL(data.video);
		url.searchParams.set('download', '1');
		return url.href;
	});
</script>

<main class="page">
	<nav class="toolbar">
		{#if data.mode === 'video'}
			<a href={videoDownload} download rel="external">
				<DownloadIcon aria-hidden="true" />{m.Download_video()}
			</a>
		{:else}
			<a href={data.screenshot} target="_blank" rel="external noopener"
				>{m.Open_original_file()}</a
			>
		{/if}
	</nav>
	<div class="columns" style:--aspect={aspect}>
		{#if data.mode === 'video'}
			<!-- Screen recording without audio; the commands are shown in the box beside it. -->
			<!-- svelte-ignore a11y_media_has_caption -->
			<video
				class="media"
				controls
				playsinline
				src={data.video}
				{@attach trackVideo}
				onloadedmetadata={(event) => {
					const video = event.currentTarget;
					if (video.videoWidth && video.videoHeight)
						aspect = video.videoWidth / video.videoHeight;
				}}
				onplay={() => {
					cancelAnimationFrame(frame);
					frame = requestAnimationFrame(followPlayback);
				}}
				onseeked={(event) => (currentTime = event.currentTarget.currentTime)}
				onpause={(event) => (currentTime = event.currentTarget.currentTime)}
			></video>
		{:else}
			<img
				class="media"
				src={data.screenshot}
				alt={m.pipeline_artifact_screenshot_tooltip()}
				onload={(event) => {
					const image = event.currentTarget as HTMLImageElement;
					if (image.naturalWidth && image.naturalHeight)
						aspect = image.naturalWidth / image.naturalHeight;
				}}
			/>
		{/if}
		<div class="box">
			{#if loaded}
				<MaestroTimelineBox
					{flows}
					empty={timeline?.entries.length ? undefined : m.Wallet_actions_no_progress()}
					highlight={data.mode === 'screenshot' ? timeline?.screenshotRow : undefined}
					onSelect={startedAt === undefined ? undefined : seekTo}
				/>
			{/if}
		</div>
	</div>
</main>

<style>
	.page {
		box-sizing: border-box;
		display: flex;
		flex-direction: column;
		gap: 12px;
		min-height: 100dvh;
		padding: 16px;
	}
	.toolbar {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 16px;
		font-size: 14px;
	}
	.toolbar a {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		color: var(--primary, inherit);
		text-decoration: underline;
	}
	.toolbar a :global(svg) {
		width: 16px;
		height: 16px;
	}
	.columns {
		--col: min(calc((100vw - 48px) / 2), calc((100dvh - 96px) * var(--aspect, 0.45)));
		display: grid;
		grid-template-columns: repeat(2, var(--col));
		justify-content: center;
		align-items: stretch;
		gap: 16px;
	}
	.media {
		display: block;
		width: 100%;
		height: auto;
		background: #000;
	}
	.box {
		contain: size;
		min-width: 0;
	}
	@media (max-width: 559px) {
		.columns {
			grid-template-columns: minmax(0, 1fr);
		}
		.media {
			width: auto;
			max-width: 100%;
			max-height: 60dvh;
			margin: 0 auto;
		}
		.box {
			min-height: 12rem;
		}
	}
</style>
