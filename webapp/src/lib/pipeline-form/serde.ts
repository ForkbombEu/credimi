// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// export async function deserializeStep(step: unknown): Promise<BuilderStep> {
// 	const parsed = serializedStepSchema.parse(step);
// 	if (parsed.type === StepType.Wallet) {
// 		const action = await pb.collection('wallet_actions').getOne(parsed.actionId);
// 		const walletItem: MarketplaceItem = await pb
// 			.collection('marketplace_items')
// 			.getFirstListItem(`type = "${Collections.Wallets}" && id = "${action.wallet}"`);
// 		const version = await pb.collection('wallet_versions').getOne(parsed.versionId);
// 		return {
// 			type: StepType.Wallet,
// 			id: action.id,
// 			name: action.name,
// 			path: walletItem.path + '/' + action.canonified_name,
// 			organization: walletItem.organization_name,
// 			yaml: action.code,
// 			recordId: action.id,
// 			data: {
// 				wallet: walletItem,
// 				version: version,
// 				action: action
// 			}
// 		};
// 	} else if (parsed.type === StepType.Credential) {
// 		return {
// 			type: StepType.Credential,
// 			recordId: parsed.recordId
// 		};
// 	} else if (parsed.type === StepType.CustomCheck) {
// 		return {
// 			type: StepType.CustomCheck,
// 			recordId: parsed.recordId
// 		};
// 	} else if (parsed.type === StepType.UseCaseVerification) {
// 		return {
// 			type: StepType.UseCaseVerification,
// 			recordId: parsed.recordId
// 		};
// 	} else {
// 		throw new Error('Invalid step type');
// 	}
// }

// /* Serialization */

// export const serializedStepSchema = z
// 	.object({
// 		type: z.union([
// 			z.literal(StepType.UseCaseVerification),
// 			z.literal(StepType.CustomCheck),
// 			z.literal(StepType.Credential)
// 		]),
// 		recordId: z.string()
// 	})
// 	.or(
// 		z.object({
// 			type: z.literal(StepType.Wallet),
// 			actionId: z.string(),
// 			walletId: z.string(),
// 			versionId: z.string()
// 		})
// 	);
