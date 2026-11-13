// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { getUserOrganization } from '$lib/utils/index.js';

import { LoginReturnSession, loginReturnPath } from '@/auth/login-return';
import { verifyUser } from '@/auth/verifyUser';
import { WelcomeSession } from '@/auth/welcome/index.js';
import { loadFeatureFlags } from '@/features';
import { deLocalizeUrl, redirect } from '@/i18n';
import { getKeyringFromLocalStorage, matchPublicAndPrivateKeys } from '@/keypairoom/keypair';
import { getUserPublicKeys, RegenerateKeyringSession } from '@/keypairoom/utils';
import { OrganizationInviteSession } from '@/organizations/invites/index.js';

//

export const load = async ({ fetch, url }) => {
	// if (!browser) return;
	const featureFlags = await loadFeatureFlags(fetch);

	// Auth

	if (!featureFlags.AUTH) error(404);
	if (!(await verifyUser(fetch))) {
		const requested = loginReturnPath(deLocalizeUrl(url).pathname + url.search);
		if (browser && requested) LoginReturnSession.start({ path: requested });
		redirect('/login');
	}

	// Keypairoom

	if (featureFlags.KEYPAIROOM && browser) {
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

	// Return to the page requested before login

	if (browser && LoginReturnSession.isActive()) {
		const requested = loginReturnPath(LoginReturnSession.getData()?.path ?? '');
		LoginReturnSession.end();
		const current = deLocalizeUrl(url);
		if (requested && requested !== current.pathname + current.search) redirect(requested);
	}

	// Organization page, redirect to edit page if first time user

	const organization = await getUserOrganization({ fetch });
	if (!organization) error(500, { message: 'USER_MISSING_ORGANIZATION' });

	const isOrganizationNotEdited = organization.created === organization.updated;
	const organizationPagePath = '/my/organization';
	if (
		isOrganizationNotEdited &&
		WelcomeSession.isActive() &&
		deLocalizeUrl(url).pathname != organizationPagePath
	) {
		WelcomeSession.end();
		redirect(organizationPagePath + '?edit=true');
	}

	return {
		organization,
		isOrganizationNotEdited
	};
};
