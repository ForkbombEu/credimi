// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { version } from '$app/environment';
import { Conformance, Pipeline } from '$lib';

import type { UsersResponse } from '@/pocketbase/types';

import { appName } from '@/brand';
import { currentUser, pb } from '@/pocketbase';

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
const authStoreUnsubscribe = pb.authStore.onChange(() => {
	currentUser.set(pb.authStore.record as UsersResponse);
	document.cookie = pb.authStore.exportToCookie({ httpOnly: false, secure: false });
});

Conformance.Standards.Store.load({ surface: 'pipeline' });

Pipeline.Runner.Catalog.init();

window.addEventListener('pagehide', () => {
	authStoreUnsubscribe();
	Pipeline.Runner.Catalog.dispose();
});
