// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { type StandardsWithTestSuites } from '$lib/standards';
import type { CustomChecksResponse } from '@/pocketbase/types';
import { SelectTestsForm, type SelectTestsFormData } from './select-tests-form';
import { getChecksConfigFormProps, type ChecksConfigFormProps } from './checks-configs-form';

//

export type StartChecksFormProps = {
	standardsWithTestSuites: StandardsWithTestSuites;
	customChecks: CustomChecksResponse[];
};

export class StartChecksForm {
	public readonly selectTestsForm: SelectTestsForm;
	public checksConfigsFormProps: ChecksConfigFormProps | undefined;

	state: 'select-tests' | 'fill-values' = $state('select-tests');

	isLoadingData = $state(false);
	loadingError = $state<Error>();

	constructor(public readonly props: StartChecksFormProps) {
		this.selectTestsForm = new SelectTestsForm({
			standards: props.standardsWithTestSuites,
			customChecks: props.customChecks,
			onSubmit: (data) => this.handleChecksSelection(data)
		});
	}

	private async handleChecksSelection(data: SelectTestsFormData) {
		this.isLoadingData = true;
		try {
			this.checksConfigsFormProps = await getChecksConfigFormProps(
				data.standardId + '/' + data.versionId,
				data.tests
			);
			this.state = 'fill-values';
		} catch (error) {
			this.loadingError = error as Error;
		} finally {
			this.isLoadingData = false;
		}
	}

	backToSelectTests() {
		this.state = 'select-tests';
		this.checksConfigsFormProps = undefined;
		this.loadingError = undefined;
	}

	submit() {
		// const form = createForm({
		// 	adapter: zod(createTestListInputSchema(data)),
		// 	onSubmit: async ({ form }) => {
		// 		const custom = customChecks.map((c) => {
		// 			return { format: 'custom', data: c.yaml };
		// 		});
		// 		await pb.send(`/api/compliance/${testId}/save-variables-and-start`, {
		// 			method: 'POST',
		// 			body: { ...form.data, ...custom }
		// 		});
		// 		await goto(`/my/tests/runs`);
		// 	},
		// 	options: {
		// 		resetForm: false
		// 	}
		// });
	}
}
