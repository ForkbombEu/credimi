// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export type { Standard, Suite, Version } from './types';
export type {
	ConformanceCheckRecord,
	ConformanceSuiteRecord,
	TemplateSurface
} from './record';
export type {
	CatalogFacets,
	SuiteFacets,
	ListAllResponse,
	StandardsWithTestSuites
} from './client';

export {
	listAll,
	getStandardsWithTestSuites,
	listChecks,
	listSuites,
	listFcafTests,
	getFcafTests,
	FCAF_STANDARD,
	fcafTestIdFromPath,
	toFcafCatalogEntry
} from './client';
export { displayNameFromUid, displayStandardName, nestSuites, titleForCheckPath } from './nest';
export { CONFORMANCE_CHECKS_COLLECTION, CONFORMANCE_SUITES_COLLECTION } from './record';

export * as Check from './check.js';
export * as Standards from './standard/index.js';
