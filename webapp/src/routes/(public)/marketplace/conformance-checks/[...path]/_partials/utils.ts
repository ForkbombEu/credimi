// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getChecksConfigsFields } from '$lib/start-checks-form/_utils';
import { startChecks, type StartChecksData } from '$lib/start-checks-form/configure-checks-form';
import { Record } from 'effect';

//

export async function startCheck(
	standardUid: string,
	versionUid: string,
	suiteUid: string,
	file: string,
	options = { fetch }
) {
	try {
		const standardAndVersionPath = `${standardUid}/${versionUid}`;

		const { specific_fields } = await getChecksConfigsFields(
			standardAndVersionPath,
			[`${suiteUid}/${file}.yaml`],
			options
		);

		const configs_with_fields: ConfigsWithFields = {};
		for (const [key, value] of Record.toEntries(specific_fields)) {
			configs_with_fields[key] = value.fields.map((field) => ({
				credimi_id: field.credimi_id,
				value: field.field_default_value ?? '',
				field_name: field.field_id
			}));
		}

		const workflowsResponse = await startChecks(
			standardAndVersionPath,
			{
				configs_with_fields,
				configs_with_json: {},
				custom_checks: {}
			},
			options
		);

		const result = workflowsResponse.results.at(0);
		if (!result) throw new Error('No result found');

		return {
			workflowId: result.workflowId,
			runId: result.workflowRunId
		};
	} catch (error) {
		return error as Error;
	}
}

type ConfigsWithFields = StartChecksData['configs_with_fields'];
