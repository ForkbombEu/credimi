// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import fs from 'node:fs';
import path from 'node:path';
import openapiTS, { astToString } from 'openapi-typescript';

import { GENERATED } from '@/utils/codegen';

//

// In openapi-typescript v7, file inputs must be provided as a URL.
// A plain string is treated as inline YAML/JSON, not a filesystem path.
const schemaUrl = new URL('../docs/public/API/openapi.yml', `file://${process.cwd()}/`);
const ast = await openapiTS(schemaUrl);
const contents = astToString(ast);

fs.writeFileSync(path.join(import.meta.dirname, `client-types.${GENERATED}.ts`), contents);
