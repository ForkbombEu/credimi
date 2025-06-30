// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import {
	type ConfigField,
	namedConfigFieldSchema,
	type NamedConfigField,
	checksConfigFieldsResponseSchema
} from '$start-checks-form/types';
import { appName } from '@/brand';
import { pb } from '@/pocketbase';
import { createStorageHandlers } from '@/utils/storage';

//

export interface BaseEditor {
	getData(): unknown;
	isValid: boolean;
}

//

export const DEFAULT_INDENTATION = 2;

export function formatJson(json: string, indentation: number = DEFAULT_INDENTATION) {
	try {
		const parsed = JSON.parse(json);
		return JSON.stringify(parsed, null, indentation);
	} catch {
		return json;
	}
}

//

export function isNamedConfigField(field: ConfigField): field is NamedConfigField {
	return namedConfigFieldSchema.safeParse(field).success;
}

export function configFieldComparator(a: ConfigField, b: ConfigField) {
	// First compare by type - string comes before object
	if (a.Type !== b.Type) {
		return a.Type === 'string' ? -1 : 1;
	}
	// Then compare by name
	return a.CredimiID.localeCompare(b.CredimiID);
}

//

export async function getChecksConfigsFields(suiteAndVersionPath: string, filenames: string[]) {
	const data = await pb.send('/api/template/placeholders', {
		method: 'POST',
		body: {
			test_id: suiteAndVersionPath,
			filenames
		}
	});
	return checksConfigFieldsResponseSchema.parse(data);
}

//

export const LatestCheckRunsStorage = createStorageHandlers(
	`${appName}-latestCheckRuns`,
	localStorage
);
