// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package evidence

import (
	"container/list"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
)

type DecoderCache struct {
	mu       sync.Mutex
	capacity int
	items    map[string]*list.Element
	order    *list.List
}

type decoderCacheEntry struct {
	key   string
	value any
}

func NewDecoderCache(capacity int) *DecoderCache {
	if capacity < 1 {
		capacity = 1
	}
	return &DecoderCache{
		capacity: capacity,
		items:    make(map[string]*list.Element, capacity),
		order:    list.New(),
	}
}

func (c *DecoderCache) Extract(root any, path string, decoder string) (any, error) {
	raw, err := extractRaw(root, path)
	if err != nil {
		return nil, err
	}
	if !isReusableCredentialDecoder(decoder) {
		return decode(raw, decoder)
	}
	key, err := decoderCacheKey(decoder, raw)
	if err != nil {
		return nil, err
	}
	if value, ok := c.get(key); ok {
		return value, nil
	}
	value, err := decode(raw, decoder)
	if err != nil {
		return nil, err
	}
	c.put(key, value)
	return value, nil
}

func isReusableCredentialDecoder(decoder string) bool {
	switch decoder {
	case "sdjwt.presentation",
		"sdjwt.presentations",
		"sdjwt.vp_token_json",
		"mdoc.presentation",
		"mdoc.vp_token_json":
		return true
	default:
		return false
	}
}

func decoderCacheKey(decoder string, raw any) (string, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return "", fmt.Errorf("marshal decoder cache input: %w", err)
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%s:%x", decoder, sum), nil
}

func (c *DecoderCache) get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	element, ok := c.items[key]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(element)
	return element.Value.(decoderCacheEntry).value, true
}

func (c *DecoderCache) put(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if element, ok := c.items[key]; ok {
		element.Value = decoderCacheEntry{key: key, value: value}
		c.order.MoveToFront(element)
		return
	}
	element := c.order.PushFront(decoderCacheEntry{key: key, value: value})
	c.items[key] = element
	if c.order.Len() <= c.capacity {
		return
	}
	oldest := c.order.Back()
	delete(c.items, oldest.Value.(decoderCacheEntry).key)
	c.order.Remove(oldest)
}
