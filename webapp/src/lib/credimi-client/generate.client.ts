// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import fs from 'node:fs';
import path from 'node:path';
import openapiTS, { astToString } from 'openapi-typescript';

import { GENERATED } from '@/utils/codegen';

//

const ast = await openapiTS(path.join(process.cwd(), '../docs/public/API/openapi.yml'));
const contents = astToString(ast);

fs.writeFileSync(path.join(import.meta.dirname, `client-types.${GENERATED}.ts`), contents);
