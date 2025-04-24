// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { currentUser, pb, type AuthStoreModel } from '@/pocketbase';

import { appVersion } from '@/utils/appVersion';
import { appName } from '@/brand';

pb.authStore.loadFromCookie(document.cookie);
pb.authStore.onChange(() => {
	currentUser.set(pb.authStore.model as AuthStoreModel);
	document.cookie = pb.authStore.exportToCookie({ httpOnly: false, secure: false });
});

console.info(
	`%c${appName} version: 🔖 ${appVersion}`,
	'font-size:4em;background: #833ab4;background:linear-gradient(to left,#833ab4,#fd1d1d,#fcb045);color:#fff;padding:4px;border-radius:4px;'
);
console.info(
	'%cmade with ❤️‍🔥 by FORKBOMB hackers',
	'font-size:2em;background:#1C39BB;color:#fff;padding:4px;border-radius:4px;'
);
