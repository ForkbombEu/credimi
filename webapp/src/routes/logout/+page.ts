// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { browser } from '$app/environment';
import { userOrganization } from '$lib/app-state';
import { disablePipelineNotifications } from '$lib/pipeline/web-push';

import { redirect } from '@/i18n';
import { currentUser, pb } from '@/pocketbase';

//

export const load = async () => {
	if (!browser) return;

	// Drop this browser's push subscription while the session can still
	// delete its record, so the next user does not get this user's notifications.
	await disablePipelineNotifications();
	localStorage.clear();
	pb.authStore.clear();
	currentUser.set(null);
	userOrganization.current = undefined;

	redirect('/');
};
