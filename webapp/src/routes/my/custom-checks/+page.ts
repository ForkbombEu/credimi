// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';

export const load = async ({ parent }) => {
    const { organization } = await parent();
    if (!organization) throw error(500, { message: 'USER_MISSING_ORGANIZATION' });

    return { organization };
};
