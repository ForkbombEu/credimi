// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { compileFromFile } from 'json-schema-to-typescript';
import fs from 'node:fs';

const map: [string, string][] = [
	[
		'../schemas/credentialissuer/openid-credential-issuer.schema.json',
		'./src/lib/types/openid-credential-issuer.generated.ts'
	],
	[
		'../schemas/pipeline/pipeline_schema.json',
		'./src/routes/my/pipelines/[action]/logic/types.generated.ts'
	]
];

for (const [schemaPath, outputPath] of map) {
	const compiled = await compileFromFile(schemaPath);
	fs.writeFileSync(outputPath, compiled);
}
