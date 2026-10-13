// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// pollTimer drives a signal-controlled polling loop. At most one interval
// timer is pending at a time, stopping cancels it, and a cancelled timer never
// requests a poll.
type pollTimer struct {
	ctx      workflow.Context
	selector workflow.Selector
	interval time.Duration

	polling bool
	tick    bool
	active  bool
	cancel  workflow.CancelFunc
	gen     int
}

// newPollTimer also adds ctx.Done() to selector, so Select returns once the
// workflow is cancelled; callers check ctx.Err() after Select.
func newPollTimer(
	ctx workflow.Context,
	selector workflow.Selector,
	interval time.Duration,
) *pollTimer {
	selector.AddReceive(ctx.Done(), func(workflow.ReceiveChannel, bool) {})
	return &pollTimer{ctx: ctx, selector: selector, interval: interval}
}

// start begins polling when it is not already running. pollNow requests a
// poll right away instead of waiting for the first interval.
func (p *pollTimer) start(pollNow bool) {
	if p.polling {
		return
	}
	p.polling = true
	if pollNow {
		p.tick = true
	}
	p.startTimer()
}

// stop pauses polling and cancels the pending timer.
func (p *pollTimer) stop() {
	p.polling = false
	if p.active {
		p.cancel()
		p.active = false
	}
}

// takeTick reports whether a poll is due and clears the request.
func (p *pollTimer) takeTick() bool {
	tick := p.tick
	p.tick = false
	return tick
}

func (p *pollTimer) startTimer() {
	if p.active {
		return
	}
	timerCtx, cancel := workflow.WithCancel(p.ctx)
	p.cancel = cancel
	p.active = true
	p.gen++
	gen := p.gen
	p.selector.AddFuture(workflow.NewTimer(timerCtx, p.interval), func(f workflow.Future) {
		if gen != p.gen {
			// A timer cancelled by stop; a newer timer may already be pending.
			return
		}
		p.active = false
		if f.Get(p.ctx, nil) == nil && p.polling {
			p.tick = true
			p.startTimer()
		}
	})
}
