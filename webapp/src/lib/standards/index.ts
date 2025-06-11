// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Either } from 'effect';
import { getStandardsAndTestSuites } from '../../routes/my/tests/new/_partials/standards-response-schema';
import type { SelectOption } from '@/components/ui-custom/utils';

export * from '../../routes/my/tests/new/_partials/standards-response-schema';

export async function getStandardsAndVersionsFlatOptionsList(): Promise<SelectOption<string>[]> {
	const response = await getStandardsAndTestSuites();
	if (!Either.isRight(response)) return [];
	const standards = response.right;
	return standards.flatMap((standard) =>
		standard.versions.map((version) => ({
			value: `${standard.uid}/${version.uid}`,
			label: `${standard.name} – ${version.name}`
		}))
	);
}
