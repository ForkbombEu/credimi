// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import {
	getStandardCheckUrl,
	getStandardCheckUrlFromJoined,
	getSuitePageUrl,
	marketplaceConformanceChecksPath
} from './urls';

//

export const Conformance = {
	basePath: marketplaceConformanceChecksPath,
	getSuitePageUrl,
	getStandardCheckUrl,
	getStandardCheckUrlFromJoined
};
