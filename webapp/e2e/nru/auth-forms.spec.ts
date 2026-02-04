// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { expect, test } from '@playwright/test';

test('login page shows required fields', async ({ page }) => {
	await page.goto('/login');
	await expect(page).toHaveURL(/login/);

	await expect(page.getByPlaceholder('name@foundation.org')).toBeVisible();
	await expect(page.getByPlaceholder('•••••')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Log in' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Forgot password' })).toBeVisible();
});

test('forgot password page shows email field', async ({ page }) => {
	await page.goto('/forgot-password');
	await expect(page).toHaveURL(/forgot-password/);

	await expect(page.getByPlaceholder('name@example.org')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Recover password' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Back' })).toBeVisible();
});
