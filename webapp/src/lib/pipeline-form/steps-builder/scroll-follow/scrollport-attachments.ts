// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Attachment } from 'svelte/attachments';

export function composeAttachments(
	...parts: Array<Attachment | undefined>
): Attachment | undefined {
	const active = parts.filter((part): part is Attachment => part != null);
	if (active.length === 0) return undefined;
	if (active.length === 1) return active[0];
	return (node) => {
		const cleanups = active
			.map((attach) => attach(node))
			.filter((cleanup): cleanup is () => void => typeof cleanup === 'function');
		if (cleanups.length === 0) return;
		return () => {
			for (const cleanup of cleanups) cleanup();
		};
	};
}

/**
 * ResizeObserver reports ~30% of scrollport height as end pad so the last short
 * card/block can scroll to center.
 */
export function endPadAttach(
	setPx: (px: number) => void,
	setViewportPx?: (px: number) => void
): Attachment {
	return (el) => {
		const update = () => {
			setPx(Math.round(el.clientHeight * 0.3));
			setViewportPx?.(el.clientHeight);
		};
		update();
		const ro = new ResizeObserver(update);
		ro.observe(el);
		return () => {
			ro.disconnect();
			setPx(0);
			setViewportPx?.(0);
		};
	};
}
