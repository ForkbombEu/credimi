// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { paraglideVitePlugin } from '@inlang/paraglide-js';
import { defineConfig } from 'vitest/config';
import { sveltekit } from '@sveltejs/kit/vite';

// These are needed for the json_typegen_wasm plugin
import wasm from 'vite-plugin-wasm';
import topLevelAwait from 'vite-plugin-top-level-await';

export default defineConfig({
	plugins: [
		wasm(),
		topLevelAwait(),
		sveltekit(),
		paraglideVitePlugin({
			project: './project.inlang',
			outdir: './src/modules/i18n/paraglide',
			strategy: ['url', 'cookie', 'baseLocale']
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
