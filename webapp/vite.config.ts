// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { paraglideVitePlugin } from '@inlang/paraglide-js';
import { defineConfig } from 'vitest/config';
import { sveltekit } from '@sveltejs/kit/vite';
import topLevelAwait from 'vite-plugin-top-level-await';

// These are needed for the json_typegen_wasm plugin
import wasm from 'vite-plugin-wasm';

export default defineConfig({
	plugins: [
		wasm(),
		sveltekit(),
		paraglideVitePlugin({
			project: './project.inlang',
			outdir: './src/modules/i18n/paraglide',
			strategy: ['url', 'cookie', 'baseLocale']
		}),
		topLevelAwait({
			// The export name of top-level await promise for each chunk module
			promiseExportName: '__tla',
			// The function to generate import names of top-level await promise in each chunk module
			promiseImportName: (i) => `__tla_${i}`
		})
	],
	optimizeDeps: {
		include: ['date-fns', 'date-fns-tz'],
		exclude: [
			'svelte-codemirror-editor',
			'codemirror',
			'@codemirror/language-javascript',
			'@codemirror/lang-json',
			'@codemirror/state',
			'thememirror'
		]
	},
	test: {
		include: ['src/**/*.{test,spec}.{js,ts}']
	},
	server: {
		port: Number(process.env.PORT) || 5100
	},
	preview: {
		allowedHosts: true
	}
});
