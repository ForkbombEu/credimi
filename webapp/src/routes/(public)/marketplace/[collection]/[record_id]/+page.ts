// SPDX-FileCopyrightText: 2025 Forkbomb BV

// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import { PocketbaseQueryAgent, type PocketbaseQueryResponse } from '@/pocketbase/query/index.js';
import type { VerifiersResponse, WalletsResponse } from '@/pocketbase/types/index.generated.js';
import { getExceptionMessage } from '@/utils/errors.js';
import { error } from '@sveltejs/kit';
import { Effect as _, pipe, Either } from 'effect';
import type { MarketplaceItem } from '../../_utils';

//

export const load = async ({ params }) => {
	const { collection, record_id } = params;

	const x = await pipe(
		_.all([
			getMarketplaceItemRecord(record_id),
			getDetailRecordPromise(collection, record_id).pipe(
				_.flatMap((r) =>
					_.tryPromise({
						try: () => r,
						catch: (e) => new DetailRecordNotFoundError(getExceptionMessage(e))
					})
				)
			)
		]),
		_.either,
		_.runPromise
	);

	if (Either.isLeft(x)) {
		error(404, {
			message: x.left.message
		});
	} else {
		return {
			marketplaceItem: x.right[0],
			detailRecord: x.right[1]
		};
	}
};

//

class MarketplaceRecordNotFoundError extends Error {}

function getMarketplaceItemRecord(record_id: string) {
	return _.tryPromise({
		try: () => pb.collection('marketplace_items').getOne(record_id) as Promise<MarketplaceItem>,
		catch: (e) => new MarketplaceRecordNotFoundError(getExceptionMessage(e))
	});
}

//

type DetailRecord =
	| VerifiersResponse
	| WalletsResponse
	| PocketbaseQueryResponse<'credential_issuers', ['credentials_via_credential_issuer']>;

class InvalidCollectionError extends Error {}

function getDetailRecordPromise(
	collection: string,
	record_id: string
): _.Effect<Promise<DetailRecord>, InvalidCollectionError> {
	let recordPromise: Promise<DetailRecord>;

	switch (collection) {
		case 'credential_issuers':
			recordPromise = new PocketbaseQueryAgent(
				{
					collection,
					expand: ['credentials_via_credential_issuer']
				},
				{ fetch }
			).getOne(record_id);
			break;
		case 'verifiers':
			recordPromise = pb.collection(collection).getOne(record_id);
			break;
		case 'wallets':
			recordPromise = pb.collection(collection).getOne(record_id);
			break;
		default:
			return _.fail(new InvalidCollectionError(`Invalid collection: ${collection}`));
	}

	return _.succeed(recordPromise);
}

//

class DetailRecordNotFoundError extends Error {}
