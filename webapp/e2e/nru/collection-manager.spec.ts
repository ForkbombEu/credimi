// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { test, expect, type Page } from '@playwright/test';
import { userLogin } from '@utils/login';
import { config } from 'dotenv';
import { nanoid } from 'nanoid';

config();

test.describe('it should test the collection manager', () => {
	let page: Page;
	const recordId = nanoid(5);
	const recordName = `test-${recordId}`;

	test.beforeAll(async ({ browser }) => {
		const context = await browser.newContext();
		page = await context.newPage();
		await page.goto('/my');
	});

	test.afterAll(async ({ browser }) => {
		await browser.close();
	});

	test('userA should login and visit the collection manager page', async ({ browser }) => {
		page = await userLogin(browser, 'A');
		await page.goto('/ui-tests/collection-manager');
	});

	test('it should create a new record', async () => {
		await page.getByRole('button', { name: 'Create record' }).click();
		await page.getByPlaceholder('abc').first().fill(recordName);
		await page.getByLabel('Number_field *').click();
		await page.getByLabel('Number_field *').press('ArrowUp');
		await page.getByLabel('Number_field *').press('ArrowUp');
		await page.getByLabel('Richtext_field *').click();
		await page.getByLabel('Richtext_field *').fill('okok');
		await page.getByLabel('Relation_field *').click();
		await page.getByRole('option').nth(0).click();
		await page.getByLabel('Relation_multi_field *').click();
		await page.getByRole('option').nth(0).click();

		// const fileChooserPromise = page.waitForEvent('filechooser');
		// await page.getByLabel('File_field *').click();
		// const fileChooser = await fileChooserPromise;
		// await fileChooser.setFiles({
		// 	name: 'test.txt',
		// 	mimeType: 'text/plain',
		// 	buffer: Buffer.from('text content')
		// });

		const submitButton = page
			.getByRole('dialog')
			.getByRole('button', { name: 'Create record' });
		await expect(submitButton).toBeEnabled();
		await submitButton.click();

		// await expect(page.getByRole('dialog')).toBeHidden();
		// await expect(page.getByText(recordName).first()).toBeVisible();
		// await expect(page.getByRole('status')).toBeVisible();
	});
});
