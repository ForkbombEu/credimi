// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package avdpool

import "time"

const (
	DefaultMaxConcurrentEmulators = 3
	DefaultMaxQueueDepth          = 50
	DefaultLeaseTimeout           = 15 * time.Minute
	DefaultHeartbeatInterval      = 30 * time.Second
)

type PoolRequest struct {
	WorkflowID  string        `json:"workflow_id"`
	RunID       string        `json:"run_id"`
	RequestID   string        `json:"request_id"`
	RequestTime time.Time     `json:"request_time"`
	Timeout     time.Duration `json:"timeout"`
	Priority    int           `json:"priority"`
}

type PoolLease struct {
	WorkflowID    string         `json:"workflow_id"`
	RunID         string         `json:"run_id"`
	RequestID     string         `json:"request_id"`
	AcquiredAt    time.Time      `json:"acquired_at"`
	LastHeartbeat time.Time      `json:"last_heartbeat"`
	EmulatorInfo  map[string]any `json:"emulator_info,omitempty"`
}

type PoolConfig struct {
	MaxConcurrentEmulators int           `json:"max_concurrent_emulators"`
	MaxQueueDepth          int           `json:"max_queue_depth"`
	LeaseTimeout           time.Duration `json:"lease_timeout"`
	HeartbeatInterval      time.Duration `json:"heartbeat_interval"`
}

type PoolState struct {
	AvailableSlots int
	QueuedRequests []PoolRequest
	ActiveLeases   map[string]PoolLease
}

type PoolManagerWorkflowInput struct {
	Config PoolConfig `json:"config,omitempty"`
	State  PoolState  `json:"state,omitempty"`
}

type PoolStatus struct {
	Available   int `json:"available"`
	Queued      int `json:"queued"`
	Active      int `json:"active"`
	MaxCapacity int `json:"max_capacity"`
}

type PoolSlotResponse struct {
	WorkflowID   string `json:"workflow_id"`
	RunID        string `json:"run_id"`
	RequestID    string `json:"request_id"`
	Granted      bool   `json:"granted"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type PoolHeartbeat struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
}

type PoolRelease struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
}

type PoolCapacityUpdate struct {
	MaxConcurrentEmulators int `json:"max_concurrent_emulators"`
}

func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxConcurrentEmulators: DefaultMaxConcurrentEmulators,
		MaxQueueDepth:          DefaultMaxQueueDepth,
		LeaseTimeout:           DefaultLeaseTimeout,
		HeartbeatInterval:      DefaultHeartbeatInterval,
	}
}

func ApplyPoolConfigDefaults(config PoolConfig) PoolConfig {
	defaults := DefaultPoolConfig()
	if config.MaxConcurrentEmulators <= 0 {
		config.MaxConcurrentEmulators = defaults.MaxConcurrentEmulators
	}
	if config.MaxQueueDepth <= 0 {
		config.MaxQueueDepth = defaults.MaxQueueDepth
	}
	if config.LeaseTimeout <= 0 {
		config.LeaseTimeout = defaults.LeaseTimeout
	}
	if config.HeartbeatInterval <= 0 {
		config.HeartbeatInterval = defaults.HeartbeatInterval
	}
	return config
}
