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
			$routes: './src/routes',
			$marketplace: './src/routes/(public)/marketplace',
			'$services-and-products': './src/routes/my/services-and-products',
			'$start-checks-form': './src/lib/start-checks-form',
			'$wallet-test': './src/routes/(public)/tests/wallet'
		},
		version: { name: process.env.npm_package_version }
	}
};

export default config;
