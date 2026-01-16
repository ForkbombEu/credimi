// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import fs from 'fs';
import adapter from 'svelte-adapter-bun';

//

const version = fs.readFileSync('../VERSION', 'utf-8').trim();

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
		version: { name: version }
	}
};

export default config;
