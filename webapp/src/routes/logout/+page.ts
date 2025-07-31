// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { browser } from '$app/environment';
import { currentUser, pb } from '@/pocketbase';
import { redirect } from '@/i18n';
import { userOrganization } from '$lib/app-state';

//

export const load = async () => {
	if (!browser) return;

	localStorage.clear();
	pb.authStore.clear();
	currentUser.set(null);
	userOrganization.current = undefined;

	redirect('/');
};
