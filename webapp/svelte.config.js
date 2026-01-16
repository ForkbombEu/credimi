// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import fs from 'fs';
import path from 'path';
import adapter from 'svelte-adapter-bun';
import { fileURLToPath } from 'url';

//

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const version = fs.readFileSync(path.join(__dirname, '../VERSION'), 'utf-8').trim();

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
