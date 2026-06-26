<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount, tick, untrack, type Snippet } from 'svelte';

	import { cn } from '@/components/ui/utils.js';

	type Props = {
		class?: string;
		contentClass?: string;
		/** Change this when scrollable content size may have changed. */
		refresh?: unknown;
		children?: Snippet;
	};

	const MIN_THUMB_PX = 48;
	/** Matches horizontal inset applied to the floating scrollbar bar (Tailwind `px-4`). */
	const BAR_INSET_X = 16;
	const DEFAULT_BAR_HEIGHT = 22;

	let { class: className, contentClass, refresh, children }: Props = $props();

	const contentId = `horizontal-scroll-${crypto.randomUUID()}`;

	let contentEl = $state<HTMLDivElement | null>(null);
	let pinnedBarEl = $state<HTMLDivElement | null>(null);
	let trackEl = $state<HTMLDivElement | null>(null);

	let barVisible = $state(false);
	let barTop = $state(0);
	let barLeft = $state(0);
	let barWidth = $state(0);
	let isPinned = $state(false);
	let thumbLeft = $state(0);
	let thumbWidth = $state(MIN_THUMB_PX);
	let scrollLeft = $state(0);
	let maxScroll = $state(0);

	function clamp(value: number, min: number, max: number) {
		return Math.min(max, Math.max(min, value));
	}

	function getTrackWidth(el: HTMLDivElement) {
		if (trackEl) {
			return trackEl.getBoundingClientRect().width;
		}

		return Math.max(0, el.clientWidth - BAR_INSET_X * 2);
	}

	function measure() {
		const el = contentEl;
		if (!el) return;

		const { scrollWidth, clientWidth } = el;
		const nextMaxScroll = Math.max(0, scrollWidth - clientWidth);
		const overflow = nextMaxScroll > 0;
		const rect = el.getBoundingClientRect();
		const viewportHeight = window.innerHeight;
		const intersectsViewport = rect.bottom > 0 && rect.top < viewportHeight;

		maxScroll = nextMaxScroll;
		scrollLeft = el.scrollLeft;
		barVisible = overflow && intersectsViewport;

		if (!barVisible) {
			thumbWidth = Math.max(0, clientWidth - BAR_INSET_X * 2);
			thumbLeft = 0;
			return;
		}

		isPinned = rect.bottom > viewportHeight;

		if (isPinned) {
			const barHeight = pinnedBarEl?.offsetHeight ?? DEFAULT_BAR_HEIGHT;
			barTop = viewportHeight - barHeight;
			barLeft = rect.left + BAR_INSET_X;
			barWidth = Math.max(0, clientWidth - BAR_INSET_X * 2);
		}

		const trackWidth = getTrackWidth(el);
		thumbWidth = Math.max(MIN_THUMB_PX, (clientWidth / scrollWidth) * trackWidth);
		const thumbTravel = Math.max(0, trackWidth - thumbWidth);
		const ratio = nextMaxScroll === 0 ? 0 : el.scrollLeft / nextMaxScroll;
		thumbLeft = clamp(ratio * thumbTravel, 0, thumbTravel);
	}

	function setScrollFromRatio(ratio: number) {
		const el = contentEl;
		if (!el || maxScroll <= 0) return;

		el.scrollLeft = clamp(ratio, 0, 1) * maxScroll;
		measure();
	}

	function onContentScroll() {
		measure();
	}

	function onTrackPointerDown(event: PointerEvent) {
		const track = trackEl;
		const el = contentEl;
		if (!track || !el || maxScroll <= 0) return;
		if (event.target !== track) return;

		const rect = track.getBoundingClientRect();
		const thumbTravel = rect.width - thumbWidth;
		const clickOffset = event.clientX - rect.left - thumbWidth / 2;
		const ratio = thumbTravel <= 0 ? 0 : clickOffset / thumbTravel;

		setScrollFromRatio(ratio);
	}

	function onThumbPointerDown(event: PointerEvent) {
		event.stopPropagation();

		const el = contentEl;
		const track = trackEl;
		if (!el || !track || maxScroll <= 0) return;

		const thumb = event.currentTarget as HTMLDivElement;
		thumb.setPointerCapture(event.pointerId);

		const rect = track.getBoundingClientRect();
		const thumbTravel = rect.width - thumbWidth;
		const startX = event.clientX;
		const startThumbLeft = thumbLeft;

		const onPointerMove = (moveEvent: PointerEvent) => {
			const delta = moveEvent.clientX - startX;
			const nextThumbLeft = clamp(startThumbLeft + delta, 0, thumbTravel);
			const ratio = thumbTravel <= 0 ? 0 : nextThumbLeft / thumbTravel;
			el.scrollLeft = ratio * maxScroll;
			measure();
		};

		const onPointerUp = (upEvent: PointerEvent) => {
			thumb.releasePointerCapture(upEvent.pointerId);
			thumb.removeEventListener('pointermove', onPointerMove);
			thumb.removeEventListener('pointerup', onPointerUp);
			thumb.removeEventListener('pointercancel', onPointerUp);
		};

		thumb.addEventListener('pointermove', onPointerMove);
		thumb.addEventListener('pointerup', onPointerUp);
		thumb.addEventListener('pointercancel', onPointerUp);
	}

	function onTrackKeyDown(event: KeyboardEvent) {
		const el = contentEl;
		if (!el) return;

		const trackWidth = trackEl?.getBoundingClientRect().width ?? el.clientWidth - BAR_INSET_X * 2;
		const step = Math.max(40, trackWidth * 0.1);

		if (event.key === 'ArrowLeft') {
			event.preventDefault();
			el.scrollLeft -= step;
			measure();
		} else if (event.key === 'ArrowRight') {
			event.preventDefault();
			el.scrollLeft += step;
			measure();
		} else if (event.key === 'Home') {
			event.preventDefault();
			setScrollFromRatio(0);
		} else if (event.key === 'End') {
			event.preventDefault();
			setScrollFromRatio(1);
		}
	}

	onMount(() => {
		const controller = new AbortController();
		const { signal } = controller;

		window.addEventListener('scroll', measure, { signal, capture: true, passive: true });
		window.addEventListener('resize', measure, { signal });

		measure();

		return () => controller.abort();
	});

	$effect(() => {
		const el = contentEl;
		if (!el) return;

		const observer = new ResizeObserver(measure);

		observer.observe(el);
		for (const child of el.children) {
			observer.observe(child);
		}

		measure();

		return () => observer.disconnect();
	});

	$effect(() => {
		if (pinnedBarEl || trackEl) {
			measure();
		}
	});

	$effect(() => {
		void refresh;
		untrack(() => {
			void tick().then(measure);
		});
	});
</script>

{#snippet scrollbarTrack()}
	<div
		bind:this={trackEl}
		class="relative h-2.5 rounded-full bg-muted"
		role="scrollbar"
		aria-controls={contentId}
		aria-orientation="horizontal"
		aria-valuemin={0}
		aria-valuemax={maxScroll}
		aria-valuenow={scrollLeft}
		tabindex="0"
		onpointerdown={onTrackPointerDown}
		onkeydown={onTrackKeyDown}
	>
		<div
			aria-hidden="true"
			class="absolute top-0 h-full cursor-grab rounded-full bg-primary active:cursor-grabbing"
			style:width="{thumbWidth}px"
			style:transform="translateX({thumbLeft}px)"
			onpointerdown={onThumbPointerDown}
		></div>
	</div>
{/snippet}

<div class={cn('relative', className)}>
	<div
		id={contentId}
		bind:this={contentEl}
		data-slot="table-container"
		class={cn('scrollbar-hide w-full overflow-x-auto', contentClass)}
		onscroll={onContentScroll}
	>
		{@render children?.()}
	</div>

	{#if barVisible && !isPinned}
		<div class="border-t border-border bg-background px-4 py-1.5">
			{@render scrollbarTrack()}
		</div>
	{/if}
</div>

{#if barVisible && isPinned}
	<div
		bind:this={pinnedBarEl}
		class="pointer-events-none fixed z-40 border-t border-border bg-background py-1.5 shadow-sm"
		style:top="{barTop}px"
		style:left="{barLeft}px"
		style:width="{barWidth}px"
	>
		<div class="pointer-events-auto">
			{@render scrollbarTrack()}
		</div>
	</div>
{/if}
