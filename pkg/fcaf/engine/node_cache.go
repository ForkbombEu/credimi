// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"container/list"
	"sync"
)

type NodeResultCache struct {
	mu       sync.Mutex
	capacity int
	items    map[string]*list.Element
	order    *list.List
}

type nodeResultCacheEntry struct {
	key   string
	value NodeResult
}

func NewNodeResultCache(capacity int) *NodeResultCache {
	if capacity < 1 {
		capacity = 1
	}
	return &NodeResultCache{
		capacity: capacity,
		items:    make(map[string]*list.Element, capacity),
		order:    list.New(),
	}
}

func (c *NodeResultCache) Get(key string) (NodeResult, bool) {
	if c == nil || key == "" {
		return NodeResult{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	element, ok := c.items[key]
	if !ok {
		return NodeResult{}, false
	}
	c.order.MoveToFront(element)
	return element.Value.(nodeResultCacheEntry).value, true
}

func (c *NodeResultCache) Put(key string, value NodeResult) {
	if c == nil || key == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if element, ok := c.items[key]; ok {
		element.Value = nodeResultCacheEntry{key: key, value: value}
		c.order.MoveToFront(element)
		return
	}
	element := c.order.PushFront(nodeResultCacheEntry{key: key, value: value})
	c.items[key] = element
	if c.order.Len() <= c.capacity {
		return
	}
	oldest := c.order.Back()
	delete(c.items, oldest.Value.(nodeResultCacheEntry).key)
	c.order.Remove(oldest)
}
