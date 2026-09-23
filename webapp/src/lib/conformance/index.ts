// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export type { Standard, Suite, Version } from './types';
export type {
	ConformanceCheckRecord,
	ConformanceSuiteRecord,
	TemplateSurface
} from './record';
export type { CatalogFacets, SuiteFacets } from './client';
export type { ListAllResponse, StandardsWithTestSuites } from './query';

export { listAll, getStandardsWithTestSuites } from './query';
export { listChecks, listSuites } from './client';
export { displayNameFromUid, displayStandardName, nestChecks, titleForCheckPath } from './nest';
export { CONFORMANCE_CHECKS_COLLECTION, CONFORMANCE_SUITES_COLLECTION } from './record';
export {
	FCAF_STANDARD,
	fcafTestIdFromPath,
	toFcafCatalogEntry,
	listFcafTests,
	getFcafTests
} from './fcaf.js';

export * as Check from './check.js';
export * as Standards from './standard/index.js';
