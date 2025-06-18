
// SPDX-FileCopyrightText: 2025 Forkbomb BV
// 
// SPDX-License-Identifier: AGPL-3.0-or-later


import { pb } from '@/pocketbase/index.js';
import { z } from 'zod';

export const WorkflowLogEntrySchema = z.object({
  _id: z.string(),
  msg: z.string(),
  src: z.string(),
  time: z.number().optional(),
  result: z
    .enum(['SUCCESS', 'ERROR', 'FAILED', 'WARNING', 'INFO'])
    .optional(),
}).passthrough();

export type WorkflowLogEntry = z.infer<typeof WorkflowLogEntrySchema>;

export interface WorkflowLogHandlers {
  onMount: () => void | Promise<void>;
  onDestroy: () => void;
}

type HandlerOptions = {
  workflowId: string;
  namespace: string;
  subscriptionSuffix: string;
  workflowSignalSuffix?: string;
  startSignal: string;
  stopSignal: string;
  onUpdate: (data: WorkflowLogEntry[]) => void;
};

export function createWorkflowLogHandlers({
  workflowId,
  namespace,
  subscriptionSuffix,
  workflowSignalSuffix,
  startSignal,
  stopSignal,
  onUpdate,
}: HandlerOptions): WorkflowLogHandlers {
  const channel = `${workflowId}${subscriptionSuffix}`;
  const signalWorkflowId = workflowSignalSuffix
    ? `${workflowId}${workflowSignalSuffix}`
    : workflowId;

  async function onMount() {
    try {
      await pb.realtime.subscribe(channel, onUpdate);
    } catch (e) {
      console.error('Subscription error:', e);
    }
    try {
      await pb.send('/api/compliance/send-temporal-signal', {
        method: 'POST',
        body: {
          workflow_id: signalWorkflowId,
          namespace,
          signal: startSignal,
        },
      });
    } catch (e) {
      console.error('Start signal error:', e);
    }
  }

  function onDestroy() {
    pb.realtime.unsubscribe(channel).catch((e) =>
      console.error('Unsubscribe error:', e)
    );
    pb.send('/api/compliance/send-temporal-signal', {
      method: 'POST',
      body: {
        workflow_id: signalWorkflowId,
        namespace,
        signal: stopSignal,
      },
    }).catch((e) => console.error('Stop signal error:', e));
  }

  return { onMount, onDestroy };
}
