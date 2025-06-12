// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import adapter from 'svelte-adapter-bun';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

const config = {
	preprocess: [vitePreprocess()],
	kit: {
		adapter: adapter(),
		alias: {
			$lib: './src/lib',
			'@': './src/modules',
			$zencode: './client_zencode',
			'$services-and-products': './src/routes/my/services-and-products'
		},
		version: { name: process.env.npm_package_version }
	}
};

export default config;
