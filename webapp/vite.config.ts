// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { paraglideVitePlugin } from '@inlang/paraglide-js';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { playwright } from '@vitest/browser-playwright';
import { loadEnv } from 'vite';
import devtoolsJson from 'vite-plugin-devtools-json';
import { defineConfig } from 'vitest/config';

export default defineConfig(({ mode }) => {
	const pocketbaseUrl = loadEnv(mode, process.cwd(), '').VITE_API;
	const proxy = pocketbaseUrl
		? { '/api': pocketbaseUrl, '/_/': pocketbaseUrl, '/temporal-ui': pocketbaseUrl }
		: undefined;
	return {
		plugins: [
			tailwindcss(),
			sveltekit(),
			devtoolsJson(),
			paraglideVitePlugin({
				project: './project.inlang',
				outdir: './src/modules/i18n/paraglide',
				strategy: ['url', 'cookie', 'baseLocale']
			})
		],

		server: {
			port: Number(process.env.PORT) || 5100,
			// The webapp talks to PocketBase on its own origin: /api, the admin assets under /_/
			// and the embedded Temporal UI (pkg/internal/temporalui) are proxied to VITE_API.
			proxy
		},
		preview: {
			allowedHosts: true,
			proxy
		},

		test: {
			expect: { requireAssertions: true },

			projects: [
				{
					extends: './vite.config.ts',

					test: {
						name: 'client',

						browser: {
							enabled: true,
							provider: playwright(),
							instances: [{ browser: 'chromium', headless: true }]
						},

						include: ['src/**/*.svelte.{test,spec}.{js,ts}'],
						exclude: ['src/lib/server/**']
					}
				},

				{
					extends: './vite.config.ts',

					test: {
						name: 'server',
						environment: 'node',
						include: ['src/**/*.{test,spec}.{js,ts}'],
						exclude: ['src/**/*.svelte.{test,spec}.{js,ts}']
					}
				}
			]
		}
	};
});
