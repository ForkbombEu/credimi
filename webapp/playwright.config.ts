// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineConfig } from '@playwright/test';

export const storageState = 'test-results/.auth/user.json';

export default defineConfig({
	webServer: {
		command: 'bash ./scripts/e2e-webserver.sh',
		port: 5100,
		timeout: 120 * 1000
	},

	testDir: 'e2e',

	projects: [
		{ name: 'setup', testMatch: /.*\.setup\.ts/ },
		{
			name: 'nruTests',
			testMatch: /nru\/.*\.spec\.ts/,
			use: {
				video: 'retain-on-failure'
			}
		},
		{
			name: 'loggedTests',
			testMatch: /logged\/.*\.spec\.ts/,
			use: {
				video: 'retain-on-failure',
				storageState
			},
			dependencies: ['setup']
		}
	]
});
