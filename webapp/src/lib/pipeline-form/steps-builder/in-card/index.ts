// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export type { InCardHostChrome } from './in-card-host-chrome.js';
export type { InCardEditMode } from './in-card-edit.js';
export type { InCardPhase, InCardSession } from './in-card-session.svelte.js';
export type { InCardLayoutEls, InCardLayoutState } from './in-card-host-machine.svelte.js';
export type { MotionHandle } from './in-card-motion.js';

export { editingUnit, isInCardEdit } from './in-card-edit.js';
export {
	createInCardSession,
	isInCardExpandReady,
	isInCardStill
} from './in-card-session.svelte.js';
export {
	inCardFormHostClass,
	playInCardEnterLayout,
	playInCardExitLayout
} from './in-card-layout.js';
export { createInCardLayoutMachine } from './in-card-host-machine.svelte.js';
export { cancelMotion, playInCardEnter, playInCardExit } from './in-card-motion.js';
export { completeInCardExit, shouldCompleteExitOnDestroy, useCardShell } from './use-card-shell.svelte.js';
export { useHeldFormMode } from './use-held-form-mode.svelte.js';

export { default as InCardHost } from './in-card-host.svelte';
export { default as InCardFormShell } from './in-card-form-shell.svelte';
