// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import * as Column from './column';
import * as conformanceChecks from './columns/conformance-checks.svelte';
import * as credentials from './columns/credentials.svelte';
import * as customIntegrations from './columns/custom-integrations.svelte';
import * as issuers from './columns/issuers.svelte';
import * as manualScheduled from './columns/manual-scheduled.svelte';
import * as minimumRunningTime from './columns/minimum-running-time.svelte';
import * as name from './columns/name.svelte';
import * as runners from './columns/runners.svelte';
import * as totalExecutionsSuccessesPercentage from './columns/total-executions-successes-percentage.svelte';
import * as useCaseVerifications from './columns/use-case-verifications.svelte';
import * as verifiers from './columns/verifiers.svelte';
import * as wallets from './columns/wallets.svelte';

//

export const columns = [
	Column.build(name),
	Column.build(wallets),
	Column.build(issuers),
	Column.build(credentials),
	Column.build(verifiers),
	Column.build(useCaseVerifications),
	Column.build(conformanceChecks),
	Column.build(customIntegrations),
	Column.build(totalExecutionsSuccessesPercentage),
	Column.build(runners),
	Column.build(manualScheduled),
	Column.build(minimumRunningTime)
];
