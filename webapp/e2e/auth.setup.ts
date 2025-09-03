// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { test as setup } from '@playwright/test';
import { userLogin } from '@utils/login';
import { config } from 'dotenv';

import { storageState } from '../playwright.config';

config();

setup('authenticate', async ({ browser }) => {
	const page = await userLogin(browser, 'A');
	await page.context().storageState({ path: storageState });
});
