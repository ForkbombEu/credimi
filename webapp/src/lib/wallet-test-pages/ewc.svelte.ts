// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Getter } from '@/utils/types';

import { pb } from '@/pocketbase';
import { warn } from '@/utils/other';

//

export function setupEWCConnections(
	workflowId: Getter<string | undefined>,
	namespace: Getter<string | undefined>
) {
	$effect(() => {
		const _workflowId = workflowId();
		const _namespace = namespace();
		if (!_workflowId || !_namespace) return;

		function unload() {
			if (!_workflowId || !_namespace) return;
			closeEWCConnections(_workflowId, _namespace);
		}

		window.addEventListener('beforeunload', unload);

		setTimeout(() => {
			pb.send('/api/compliance/send-temporal-signal', {
				method: 'POST',
				body: {
					workflow_id: _workflowId,
					namespace: _namespace,
					signal: 'start-ewc-check-signal'
				}
			}).catch((err) => {
				warn(err);
			});
		}, 2000);

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
