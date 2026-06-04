// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { toast } from 'svelte-sonner';

import { getExceptionMessage } from '@/utils/errors';

let lastMessage = '';
let lastMessageAt = 0;

export function showPipelineFormError(error: unknown) {
	const message = getExceptionMessage(error);
	const now = Date.now();

	if (message === lastMessage && now - lastMessageAt < 1000) return;

	lastMessage = message;
	lastMessageAt = now;
	toast.error(message);
}
