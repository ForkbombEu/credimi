// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { StandardsWithTestSuites } from '$lib/standards';
import type { CustomChecksResponse } from '@/pocketbase/types';
import { String } from 'effect';
import { watch } from 'runed';
import type { ChecksConfigFieldsResponse } from '../types';
import { getChecksConfigsFields } from '$start-checks-form/_utils';

//

export type SelectChecksSubmitData = {
	standardAndVersionPath: string;
	checksConfigsFields?: ChecksConfigFieldsResponse | undefined;
	customChecks: CustomChecksResponse[];
};

export type SelectChecksFormProps = {
	standards: StandardsWithTestSuites;
	customChecks: CustomChecksResponse[];
	onSubmit: (data: SelectChecksSubmitData) => void | Promise<void>;
};

export class SelectChecksForm {
	constructor(private readonly props: SelectChecksFormProps) {
		this.registerEffect_DeselectOnStandardChange();
		this.registerEffect_DeselectOnVersionChange();
		this.registerEffect_UpdateSelectedVersion(); // Must be last!
	}

	// Selection: Standard

	availableStandards = $derived.by(() => this.props.standards);

	selectedStandardId = $state('');
	selectedStandard = $derived.by(() => {
		return this.props.standards.find((standard) => standard.uid === this.selectedStandardId);
	});

	// Selection: Version

	availableVersions = $derived(this.selectedStandard?.versions ?? []);
	hasOnlyOneVersion = $derived(this.availableVersions.length === 1);

	selectedVersionId = $state('');
	selectedVersion = $derived.by(() => {
		return this.availableVersions.find((version) => version.uid === this.selectedVersionId);
	});

	private registerEffect_UpdateSelectedVersion() {
		watch(
			() => this.availableVersions,
			(availableVersions) => {
				if (availableVersions.length === 1) {
					this.selectedVersionId = this.availableVersions[0].uid;
				}
			}
		);
	}

	// Selection: Entire suites | Individual tests

	availableSuites = $derived(this.selectedVersion?.suites ?? []);

	availableSuitesWithTests = $derived(
		this.availableSuites.filter((suite) => suite.files.length > 0)
	);
	selectedSuites = $state<string[]>([]);

	availableSuitesWithoutTests = $derived(
		this.availableSuites.filter((suite) => suite.files.length === 0)
	);
	selectedTests = $state<string[]>([]);

	selectedSuiteId = $state<string>();
	selectedSuite = $derived.by(() => {
		return this.availableSuites.find((suite) => suite.uid === this.selectedSuiteId);
	});

	// Selection: Custom checks

	availableCustomChecks = $derived.by(() => {
		return this.props.customChecks
			.filter((check) => {
				if (!this.selectedStandardId) return false;
				return check.standard_and_version.startsWith(this.selectedStandardId);
			})
			.filter((check) => {
				if (!this.selectedVersionId) return false;
				return check.standard_and_version.endsWith(this.selectedVersionId);
			});
	});

	selectedCustomChecksIds = $state<string[]>([]);
	selectedCustomChecks = $derived.by(() =>
		this.availableCustomChecks.filter((check) =>
			this.selectedCustomChecksIds.includes(check.id)
		)
	);

	// Deselect

	private registerEffect_DeselectOnStandardChange() {
		watch(
			() => this.selectedStandardId,
			() => {
				this.selectedVersionId = '';
				this.selectedSuites = [];
				this.selectedTests = [];
				this.selectedCustomChecksIds = [];
			}
		);
	}

	private registerEffect_DeselectOnVersionChange() {
		watch(
			() => this.selectedVersionId,
			() => {
				this.selectedSuites = [];
				this.selectedTests = [];
				this.selectedCustomChecksIds = [];
			}
		);
	}

	// Submission

	hasOnlyCustomChecks = $derived(
		this.selectedSuites.length === 0 &&
			this.selectedTests.length === 0 &&
			this.selectedCustomChecksIds.length > 0
	);

	hasSelection = $derived(
		this.selectedSuites.length > 0 ||
			this.selectedTests.length > 0 ||
			this.selectedCustomChecksIds.length > 0
	);

	isValid = $derived(
		String.isNonEmpty(this.selectedStandardId) &&
			String.isNonEmpty(this.selectedVersionId) &&
			this.hasSelection
	);

	isLoading = $state(false);
	loadingError = $state<Error>();

	async submit() {
		if (!this.isValid) return;
		this.isLoading = true;
		try {
			const standardAndVersionPath = this.selectedStandardId + '/' + this.selectedVersionId;

			let checksConfigsFields: ChecksConfigFieldsResponse | undefined;
			if (!this.hasOnlyCustomChecks) {
				checksConfigsFields = await getChecksConfigsFields(
					standardAndVersionPath,
					this.selectedSuites.concat(this.selectedTests)
				);
			}

			this.props.onSubmit(
				$state.snapshot({
					standardAndVersionPath,
					checksConfigsFields,
					customChecks: this.selectedCustomChecks
				})
			);
		} catch (error) {
			this.loadingError = error as Error;
		} finally {
			this.isLoading = false;
		}
	}
}
