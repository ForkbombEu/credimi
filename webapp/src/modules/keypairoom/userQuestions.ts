// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { String } from 'effect';
import _ from 'lodash';
import z from 'zod';

import { m } from '@/i18n';

//

export const userChallengesSchema = z
	.object({
		whereParentsMet: z.string(),
		nameFirstPet: z.string(),
		whereHomeTown: z.string(),
		nameFirstTeacher: z.string(),
		nameMotherMaid: z.string()
	})
	.partial()
	.refine((data) => {
		return Object.values(data).filter(isStringNonEmpty).length >= 3;
	}, 'AT_LEAST_THREE_QUESTIONS');

export type UserChallenges = z.infer<typeof userChallengesSchema>;
export type UserChallenge = keyof UserChallenges;

export const userChallenges: Array<{ id: UserChallenge; text: string }> = [
	{ id: 'whereParentsMet', text: m.whereParentsMet() },
	{ id: 'nameFirstPet', text: m.nameFirstPet() },
	{ id: 'whereHomeTown', text: m.whereHomeTown() },
	{ id: 'nameFirstTeacher', text: m.nameFirstTeacher() },
	{ id: 'nameMotherMaid', text: m.nameMotherMaid() }
];

//

const ZENROOM_NULL = 'null';

export function formatAnswersForZenroom(data: UserChallenges): UserChallenges {
	return _.mapValues(data, (v) => (isStringNonEmpty(v) ? v : ZENROOM_NULL));
}

function isStringNonEmpty(string: string | undefined) {
	if (!string) return false;
	else return String.isNonEmpty(string);
}
