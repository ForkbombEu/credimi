// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { paraglideVitePlugin } from '@inlang/paraglide-js';
import { sveltekit } from '@sveltejs/kit/vite';
// These are needed for the json_typegen_wasm plugin
import wasm from 'vite-plugin-wasm';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [
		wasm(),
		sveltekit(),
		paraglideVitePlugin({
			project: './project.inlang',
			outdir: './src/modules/i18n/paraglide',
			strategy: ['url', 'cookie', 'baseLocale']
		})
	],
	esbuild: {
		supported: {
			'top-level-await': true
		}
	},
	optimizeDeps: {
		include: ['date-fns', 'date-fns-tz'],
		exclude: [
			'svelte-codemirror-editor',
			'codemirror',
			'@codemirror/language-javascript',
			'@codemirror/lang-json',
			'@codemirror/lang-yaml',
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
