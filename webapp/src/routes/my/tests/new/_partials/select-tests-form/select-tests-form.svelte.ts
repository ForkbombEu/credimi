// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { StandardsWithTestSuites } from '$lib/standards';
import type { CustomChecksResponse } from '@/pocketbase/types';
import { String } from 'effect';
import { watch } from 'runed';

//

export type SelectTestsFormData = {
	standardId: string;
	versionId: string;
	suites: string[];
	tests: string[];
	customChecks: string[];
};

export type SelectTestsFormProps = {
	standards: StandardsWithTestSuites;
	customChecks: CustomChecksResponse[];
	onSubmit: (data: SelectTestsFormData) => void;
};

export class SelectTestsForm {
	constructor(private readonly props: SelectTestsFormProps) {
		this.effectDeselectOnStandardChange();
		this.effectDeselectOnVersionChange();
		this.effectUpdateSelectedVersion(); // Must be last!
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

	effectUpdateSelectedVersion() {
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

	selectedCustomChecks = $state<string[]>([]);

	// Deselect

	effectDeselectOnStandardChange() {
		watch(
			() => this.selectedStandardId,
			() => {
				this.selectedVersionId = '';
				this.selectedSuites = [];
				this.selectedTests = [];
				this.selectedCustomChecks = [];
			}
		);
	}

	effectDeselectOnVersionChange() {
		watch(
			() => this.selectedVersionId,
			() => {
				this.selectedSuites = [];
				this.selectedTests = [];
				this.selectedCustomChecks = [];
			}
		);
	}

	// Submission

	hasSelection = $derived(
		this.selectedSuites.length > 0 ||
			this.selectedTests.length > 0 ||
			this.selectedCustomChecks.length > 0
	);

	isValid = $derived(
		String.isNonEmpty(this.selectedStandardId) &&
			String.isNonEmpty(this.selectedVersionId) &&
			this.hasSelection
	);

	submit() {
		if (!this.isValid) return;

		this.props.onSubmit(
			$state.snapshot({
				standardId: this.selectedStandardId,
				versionId: this.selectedVersionId,
				suites: this.selectedSuites,
				tests: this.selectedTests,
				customChecks: this.selectedCustomChecks
			})
		);
	}
}
