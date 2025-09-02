// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Getter } from '@/utils/types';

import { pb } from '@/pocketbase';
import { warn } from '@/utils/other';

//

export function setupEWCConnections(workflowId: Getter<string>, namespace: Getter<string>) {
	$effect(() => {
		function unload() {
			closeEWCConnections(workflowId(), namespace());
		}

		window.addEventListener('beforeunload', unload);

		pb.send('/api/compliance/send-temporal-signal', {
			method: 'POST',
			body: {
				workflow_id: workflowId(),
				namespace: namespace(),
				signal: 'start-ewc-check-signal'
			}
		}).catch((err) => {
			warn(err);
		});

		return () => {
			window.removeEventListener('beforeunload', unload);
			unload();
		};
	});
}

function closeEWCConnections(workflowId: string, namespace: string) {
	pb.send('/api/compliance/send-temporal-signal', {
		method: 'POST',
		body: {
			workflow_id: workflowId,
			namespace: namespace,
			signal: 'stop-ewc-check-signal'
		}
	}).catch((err) => {
		warn(err);
	});
}
