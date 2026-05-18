// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import {
	getStandardCheckUrl,
	getStandardCheckUrlFromPath,
	getSuitePageUrl,
	hubConformanceChecksPath
} from './urls';

//

export const Conformance = {
	basePath: hubConformanceChecksPath,
	getSuitePageUrl,
	getStandardCheckUrl,
	getStandardCheckUrlFromPath
};
