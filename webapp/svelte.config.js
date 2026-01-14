// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import adapter from 'svelte-adapter-bun';

import { appVersion } from './src/modules/utils/appVersion';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: [vitePreprocess()],
	kit: {
		adapter: adapter(),
		alias: {
			'@': './src/modules',
			$lib: './src/lib',
			$zencode: './client_zencode',
			'$start-checks-form': './src/lib/start-checks-form',
			'$pipeline-form': './src/lib/pipeline-form',
			$root: '..'
		},
		version: { name: appVersion }
	}
};

export default config;
