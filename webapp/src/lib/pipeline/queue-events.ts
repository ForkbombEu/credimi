// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Event type for queue cancel requests.
const QUEUE_CANCEL_EVENT = 'queue-cancel-requested';

// Module-scoped event bus for queue cancel coordination.
const queueCancelEvents = new EventTarget();

// Build a queue cancel event with the ticket id payload.
const buildQueueCancelEvent = (ticketId: string): Event => {
	if (typeof CustomEvent === 'function') {
		return new CustomEvent<string>(QUEUE_CANCEL_EVENT, { detail: ticketId });
	}

	const event = new Event(QUEUE_CANCEL_EVENT);
	(event as CustomEvent<string>).detail = ticketId;
	return event;
};

// Emit a cancel request so active queue pollers can stop cleanly.
export function emitQueueCancelRequested(ticketId: string): void {
	queueCancelEvents.dispatchEvent(buildQueueCancelEvent(ticketId));
}

// Subscribe to cancel requests; returns an unsubscribe function.
export function onQueueCancelRequested(handler: (ticketId: string) => void): () => void {
	const listener = (event: Event) => {
		const detail = (event as CustomEvent<string>).detail;
		if (typeof detail === 'string') handler(detail);
	};

	queueCancelEvents.addEventListener(QUEUE_CANCEL_EVENT, listener);
	return () => queueCancelEvents.removeEventListener(QUEUE_CANCEL_EVENT, listener);
}
