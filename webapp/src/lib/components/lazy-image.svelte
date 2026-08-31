<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ClassValue } from 'svelte/elements';

	type Props = {
		src: string;
		alt?: string;
		class?: ClassValue;
		width?: number | string;
		height?: number | string;
		loading?: 'lazy' | 'eager';
		decoding?: 'async' | 'sync' | 'auto';
	};

	let {
		src,
		alt = '',
		class: className,
		width,
		height,
		loading = 'lazy',
		decoding = 'async'
	}: Props = $props();

	let element = $state<HTMLImageElement | null>(null);
	let shouldLoad = $state(false);

	/**
	 * Native `loading="lazy"` only defers against the document viewport, so images
	 * inside an `overflow-y: auto` panel (like a sheet) are not deferred correctly.
	 * Walk up to the nearest scrollable ancestor and observe that instead.
	 */
	function closestScrollable(el: HTMLElement | null): HTMLElement | null {
		let current = el?.parentElement ?? null;
		while (current) {
			const { overflowY } = getComputedStyle(current);
			if (
				(overflowY === 'auto' || overflowY === 'scroll') &&
				current.scrollHeight > current.clientHeight
			) {
				return current;
			}
			current = current.parentElement;
		}
		return null;
	}

	$effect(() => {
		const el = element;
		if (!el) return;
		if (shouldLoad) return;

		if (typeof IntersectionObserver === 'undefined') {
			shouldLoad = true;
			return;
		}

		const observer = new IntersectionObserver(
			(entries) => {
				if (entries.some((entry) => entry.isIntersecting)) {
					shouldLoad = true;
					observer.disconnect();
				}
			},
			{ root: closestScrollable(el), rootMargin: '600px 0px' }
		);
		observer.observe(el);
		return () => observer.disconnect();
	});
</script>

<img
	bind:this={element}
	src={shouldLoad ? src : undefined}
	{alt}
	{width}
	{height}
	{loading}
	{decoding}
	class={className}
/>
