// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HistoryEvent } from '@forkbombeu/temporal-ui';
import type { WorkflowExecution } from '@forkbombeu/temporal-ui/dist/types/workflows';
import type { WorkflowMemo } from '$lib/workflows';

import { browser } from '$app/environment';
import { onDestroy } from 'svelte';

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

type WorkflowMessage = Message<
	'workflow',
	{
		execution: WorkflowExecution;
		eventHistory: HistoryEvent[];
		memo?: WorkflowMemo;
	}
>;

// Iframe

export type IframeMessage = HeightMessage | ReadyMessage;

type HeightMessage = Message<
	'height',
	{
		height: number;
	}
>;

type ReadyMessage = Message<'ready'>;
