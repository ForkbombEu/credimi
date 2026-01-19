// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { browser } from '$app/environment';
import { onDestroy } from 'svelte';

import type { _getWorkflow } from '../+layout';

/* Messaging utilities */

type Message<Name extends string, Data = unknown> = Data & {
	type: Name;
};

type BaseMessage = Message<string, unknown>;

type MessageTarget = Pick<Window, 'postMessage'>;

//

export function setupEmitter<M extends BaseMessage>(target: () => MessageTarget | undefined) {
	return (event: M) => {
		target()?.postMessage(event);
	};
}

export function setupListener<M extends BaseMessage>(
	onMessage: (event: M | Message<never, never>) => void
) {
	if (!browser) return;

	function actualOnMessage(event: MessageEvent) {
		if ('type' in event.data && typeof event.data.type === 'string') {
			onMessage(event.data);
		}
	}

	window.addEventListener('message', actualOnMessage);

	onDestroy(() => {
		window.removeEventListener('message', actualOnMessage);
	});
}

/* Actual messages */

// Page

export type PageMessage = WorkflowMessage;

type WorkflowMessage = Message<'workflow', WorkflowResponse>;
type WorkflowResponse = Exclude<Awaited<ReturnType<typeof _getWorkflow>>, Error>;

// Iframe

export type IframeMessage = HeightMessage | ReadyMessage;

type HeightMessage = Message<
	'height',
	{
		height: number | undefined | null;
	}
>;

type ReadyMessage = Message<'ready'>;
