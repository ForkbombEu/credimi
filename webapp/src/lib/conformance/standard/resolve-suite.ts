// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import * as Store from '../store.svelte.js';

//

export function resolveSuite(standardUid: string, versionUid: string, suiteUid: string) {
	const std = Store.get().standards.find((s) => s.uid === standardUid);
	const ver = std?.versions.find((v) => v.uid === versionUid);
	const suite = ver?.suites.find((su) => su.uid === suiteUid);
	return {
		logo: suite?.logo,
		name: suite?.name ?? suiteUid
	};
}
