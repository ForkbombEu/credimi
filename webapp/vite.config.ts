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
	const pocketbaseUrl = loadEnv(mode, process.cwd(), '').PUBLIC_POCKETBASE_URL;
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
			// The embedded Temporal UI sends X-Frame-Options: SAMEORIGIN, so in dev it is served from
			// the webapp origin; PocketBase owns /temporal-ui (pkg/internal/temporalui).
			proxy: pocketbaseUrl ? { '/temporal-ui': new URL(pocketbaseUrl).origin } : undefined
		},
		preview: {
			allowedHosts: true
		},

		optimizeDeps: {
			include: ['date-fns', 'date-fns-tz'],
			exclude: [
				'codemirror',
				'@codemirror/language-javascript',
				'@codemirror/lang-json',
				'@codemirror/lang-yaml',
				'@codemirror/state',
				'@codemirror/lint',
				'@codemirror/search',
				'thememirror'
			]
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
