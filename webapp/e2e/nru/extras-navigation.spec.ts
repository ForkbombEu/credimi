// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { expect, test } from '@playwright/test';

const tools = [
	['Wallet recorder', 'capture-wallet', 'credimi-capture-wallet'],
	['Issuer and verifier inspector', 'capture-issuer-verifier', 'eudi-conformance-evidence'],
	['Trusted list inspector', 'trust-inspector', 'eudi-trusted-lists-inspector'],
	['Trusted list publisher', 'lote', 'eudi-trusted-list-publisher'],
	['Status list server', 'tsl', 'capture-status-list'],
	['Federation trust anchor', 'federation', 'credimi-federation'],
	['Rulebook catalogue', 'attestbook', 'credimi-attestbook'],
	['Atlas', 'atlas', 'eudi-conformance-atlas']
];

for (const width of [1440, 390]) {
	test(`reaches every Extra from the header link at ${width}px`, async ({ page }) => {
		await page.setViewportSize({ width, height: 844 });
		// Wait for hydration: the mobile menu does nothing before it.
		await page.goto('/pages/guides', { waitUntil: 'networkidle' });
		if (width === 390) await page.getByRole('button', { name: 'Menu', exact: true }).click();

		// One entry point, named Extras: no menu of bare tool names in the header.
		await page.getByRole('link', { name: 'Extras', exact: true }).click();
		await expect(page).toHaveURL(/\/pages\/extras$/);
		await expect(page.getByRole('heading', { level: 1 })).toHaveText(
			'See what your identity stack is doing.'
		);

		// The top of the page is the shortcut list: all eight, each explained, each one click
		// from its hosted app.
		const shortcuts = page.getByRole('navigation', { name: 'Open an Extra' }).getByRole('link');
		await expect(shortcuts).toHaveCount(tools.length);
		for (const [i, [name, host]] of tools.entries()) {
			const shortcut = shortcuts.nth(i);
			await expect(shortcut).toContainText(name);
			await expect(shortcut).toContainText('?');
			await expect(shortcut).toHaveAttribute('href', `https://${host}.credimi.io/`);
		}

		// Keyboard: a focused shortcut launches its tool in a new tab.
		const toolUrl = 'https://capture-wallet.credimi.io/';
		await page.context().route(toolUrl, (route) => route.fulfill({ body: 'Wallet Capture' }));
		await shortcuts.first().focus();
		const newTab = page.waitForEvent('popup');
		await page.keyboard.press('Enter');
		const tool = await newTab;
		await expect(tool).toHaveURL(toolUrl);
		await tool.close();
		await expect(page).toHaveURL(/\/pages\/extras$/);

		// Each tool's own section launches it too, and links its source.
		for (const [name, host, repo] of tools) {
			await expect(
				page.getByRole('link', { name: `Open ${name}`, exact: true })
			).toHaveAttribute('href', `https://${host}.credimi.io/`);
			await expect(
				page.getByRole('link', { name: `${name} on GitHub`, exact: true })
			).toHaveAttribute('href', `https://github.com/ForkbombEu/${repo}`);
		}

		const overflows = await page.evaluate(
			() => document.documentElement.scrollWidth > document.documentElement.clientWidth
		);
		expect(overflows).toBe(false);
	});
}
