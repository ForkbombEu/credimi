// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { copyFileSync } from 'node:fs';
import { join } from 'node:path';

export default async function setup() {
	const sourceDb = join(process.cwd(), '../fixtures/test_pb_data/data.db');
	const testDb = join('/tmp', 'credimi-webapp-test-data.db');

	copyFileSync(sourceDb, testDb);
	process.env.DATA_DB_PATH = testDb;

	const { generateCollectionsModels } =
		await import('./collections-models/generate.collections-models');
	await generateCollectionsModels();
}
