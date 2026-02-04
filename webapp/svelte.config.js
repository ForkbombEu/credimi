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
	// Consult https://svelte.dev/docs/kit/integrations
	// for more information about preprocessors
	preprocess: vitePreprocess(),

	kit: {
		// adapter-auto only supports some environments, see https://svelte.dev/docs/kit/adapter-auto for a list.
		// If your environment is not supported, or you settled on a specific environment, switch out the adapter.
		// See https://svelte.dev/docs/kit/adapters for more information about adapters.
		adapter: adapter(),
		alias: {
			'@': './src/modules',
			$zencode: './client_zencode',
			'$start-checks-form': './src/lib/start-checks-form',
			'$pipeline-form': './src/lib/pipeline-form',
			$root: '..'
		},
		version: { name: version }
	}
};

export default config;
