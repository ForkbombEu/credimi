// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

// NewsToolsTaskQueue is the default-namespace task queue serving the news
// activities that the news agent calls by name.
const NewsToolsTaskQueue = "NewsToolsTaskQueue"

// NewsAgentTaskQueue is the default-namespace task queue of the Python news
// agent worker, which runs the NewsDailyScan workflow.
const NewsAgentTaskQueue = "news-agent"

// NewsDailyScanWorkflowName is the workflow type the daily news scan schedule
// starts on NewsAgentTaskQueue.
const NewsDailyScanWorkflowName = "NewsDailyScan"

// NewsDailyScanScheduleID is the ID of the daily news scan schedule.
const NewsDailyScanScheduleID = "news-daily-scan"

// NewsDailyScanInput is the input of the NewsDailyScan workflow.
type NewsDailyScanInput struct {
	SourceIDs []string `json:"source_ids"`
	MaxDrafts int      `json:"max_drafts"`
}
