// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import { CredimiClient } from './credimiClient.generated';

const client = new CredimiClient(pb);

export { client };
