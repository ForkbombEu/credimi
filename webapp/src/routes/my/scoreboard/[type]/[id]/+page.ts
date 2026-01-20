// SPDX-FileCopyrightText: 2025 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later

import { fetchMyResults, type OTelSpan, type ScoreboardEntry } from '$lib/scoreboard';

import type { PageLoad } from './$types';

//

export const load: PageLoad = async ({ params }) => {
	const data = await fetchMyResults();
	const { type, id } = params;

	// Find the specific entry
	const allEntries = [
		...data.summary.wallets,
		...data.summary.issuers,
		...data.summary.verifiers,
		...data.summary.pipelines
	];
	const entry = allEntries.find((e: ScoreboardEntry) => e.id === id && e.type === type) || null;

	// Extract relevant spans from OTel data
	const spans: OTelSpan[] = [];
	if (data.otelData?.resourceSpans) {
		for (const rs of data.otelData.resourceSpans) {
			for (const ss of rs.scopeSpans) {
				spans.push(
					...ss.spans.filter((span: OTelSpan) =>
						span.attributes.some(
							(attr) => attr.key === 'entity.id' && attr.value === id
						)
					)
				);
			}
		}
	}

	return {
		entry,
		spans,
		type,
		id
	};
};
