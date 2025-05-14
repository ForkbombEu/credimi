// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { verifyUser } from '@/auth/verifyUser';
import { loadFeatureFlags } from '@/features';
import { error } from '@sveltejs/kit';

import { browser } from '$app/environment';
import { deLocalizeHref, redirect } from '@/i18n';
import { getKeyringFromLocalStorage, matchPublicAndPrivateKeys } from '@/keypairoom/keypair';
import { getUserPublicKeys, RegenerateKeyringSession } from '@/keypairoom/utils';

import { OrganizationInviteSession } from '@/organizations/invites/index.js';
import { getUserOrganization } from '$lib/utils';
import { WelcomeSession } from '@/auth/welcome/index.js';

//

export const load = async ({ fetch, url }) => {
	if (!browser) return;
	const featureFlags = await loadFeatureFlags(fetch);

	// Auth

	if (!featureFlags.AUTH) error(404);
	if (!(await verifyUser(fetch))) redirect('/login');

	// Keypairoom

	if (featureFlags.KEYPAIROOM) {
		const publicKeys = await getUserPublicKeys();
		if (!publicKeys) redirect('/keypairoom');

		const keyring = getKeyringFromLocalStorage();
		if (!keyring) {
			RegenerateKeyringSession.start();
			redirect('/keypairoom/regenerate');
		}

		try {
			if (publicKeys && keyring) await matchPublicAndPrivateKeys(publicKeys, keyring);
		} catch {
			RegenerateKeyringSession.start();
			redirect('/keypairoom/regenerate');
		}
	}
	if (featureFlags.KEYPAIROOM && RegenerateKeyringSession.isActive()) {
		RegenerateKeyringSession.end();
	}

	// Organizations

	if (featureFlags.ORGANIZATIONS && OrganizationInviteSession.isActive()) {
		OrganizationInviteSession.end();
		redirect('/my/organizations');
	}

	//

	const organizationData = await getUserOrganization({ fetch });
	if (!organizationData) error(500, { message: 'NO_ORGANIZATION' });

	const { organization, organizationInfo } = organizationData;
	const isOrganizationInfoMissing = organizationInfo.created === organizationInfo.updated;

	if (WelcomeSession.isActive() && deLocalizeHref(url.pathname) !== '/my/organization') {
		WelcomeSession.end();
		redirect('/my/organization?edit=true');
	}

	return {
		organization,
		organizationInfo,
		isOrganizationInfoMissing
	};
};
