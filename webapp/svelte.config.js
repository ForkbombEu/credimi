// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import adapter from 'svelte-adapter-bun';

const config = {
	preprocess: [vitePreprocess()],
	kit: {
		adapter: adapter(),
		alias: {
			'@': './src/modules',
			$lib: './src/lib',
			$zencode: './client_zencode',
			'$start-checks-form': './src/lib/start-checks-form',
			$root: '..'
		},
		version: { name: process.env.npm_package_version }
	}
};

export default config;
