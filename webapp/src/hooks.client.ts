// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HandleClientError } from '@sveltejs/kit';

import { version } from '$app/environment';
import { env } from '$env/dynamic/public';
import * as ConformanceStore from '$lib/conformance/store.svelte.js';
import * as DeviceCatalog from '$lib/pipeline/device/catalog.svelte.js';
import { syncPushSubscriptionOwner } from '$lib/pipeline/web-push';
import { reportError, startBrowserTelemetry } from '$lib/telemetry/client';

import type { UsersResponse } from '@/pocketbase/types';

import { appName } from '@/brand';
import { currentUser, pb } from '@/pocketbase';

//

if (env.PUBLIC_OTEL_BROWSER_ENABLED === 'true') {
	startBrowserTelemetry(version, () =>
		pb.authStore.isValid && pb.authStore.record ? { 'enduser.id': pb.authStore.record.id } : {}
	);
}

// Unexpected errors in load functions and components. A missing page is not an
// application error.
export const handleError: HandleClientError = ({ error, event, status }) => {
	if (status === 404) return;
	reportError(error, {
		'exception.source': 'sveltekit.handleError',
		'http.response.status_code': status,
		'sveltekit.route.id': event.route.id ?? ''
	});
};

//
console.info(
	`%c${appName} version: 🔖 ${version}`,
	'font-size:4em;background: #833ab4;background:linear-gradient(to left,#833ab4,#fd1d1d,#fcb045);color:#fff;padding:4px;border-radius:4px;'
);
console.info(
	'%cmade with ❤️‍🔥 by FORKBOMB hackers',
	'font-size:2em;background:#1C39BB;color:#fff;padding:4px;border-radius:4px;'
);

//

pb.authStore.loadFromCookie(document.cookie);

// Session expiry, sign-out and switching account all go through the auth
// store, which explicit logout alone does not cover. An expired token counts
// as signed out.
let pushSubscriptionOwner = pb.authStore.isValid ? pb.authStore.record?.id : undefined;
void syncPushSubscriptionOwner(pushSubscriptionOwner);

const authStoreUnsubscribe = pb.authStore.onChange(() => {
	currentUser.set(pb.authStore.record as UsersResponse);
	document.cookie = pb.authStore.exportToCookie({ httpOnly: false, secure: false });

	const user = pb.authStore.isValid ? pb.authStore.record?.id : undefined;
	if (user === pushSubscriptionOwner) return;
	pushSubscriptionOwner = user;
	void syncPushSubscriptionOwner(user);
});

// Not awaited: a top-level await here held back hydration of every page until the catalog
// arrived. Its readers (scoreboard suite names) are reactive and update when it lands.
ConformanceStore.load({ surface: 'pipeline' }).catch(console.error);

DeviceCatalog.init();

window.addEventListener('pagehide', () => {
	authStoreUnsubscribe();
	DeviceCatalog.dispose();
});
