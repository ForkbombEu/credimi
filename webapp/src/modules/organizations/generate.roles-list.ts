// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import fs from 'fs';
import path from 'node:path';

import type { OrgRolesRecord } from '@/pocketbase/types';

import { formatCode, GENERATED, openDb, logCodegenResult } from '@/utils/codegen';

//

const TYPE_NAME = 'OrgRole';
const OBJECT_NAME = `${TYPE_NAME}s`;

const db = await openDb();
const rolesRecords = (await db.all('SELECT * FROM orgRoles')) as OrgRolesRecord[];
const rolesEntries = rolesRecords.map((r) => `${r.name.toUpperCase()}: '${r.name}'`);

const code = `
export const ${OBJECT_NAME} = {
	${rolesEntries.join(',\n')}
} as const

export type ${TYPE_NAME} = typeof ${OBJECT_NAME} [keyof typeof ${OBJECT_NAME}];
`;

const formattedCode = await formatCode(code);
const filePath = path.join(import.meta.dirname, `roles-list.${GENERATED}.ts`);
fs.writeFileSync(filePath, formattedCode);
logCodegenResult('organization roles list', filePath);
