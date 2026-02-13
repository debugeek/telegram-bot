package tgbot

import "container/list"

type LRUMap[K comparable, V any] struct {
	maxCount int
	order    *list.List
	items    map[K]lruEntry[K, V]
}

type lruEntry[K comparable, V any] struct {
	value V
	node  *list.Element
}

func NewLRUMap[K comparable, V any](maxCount int) *LRUMap[K, V] {
	if maxCount <= 0 {
		maxCount = 1
	}
	return &LRUMap[K, V]{
		maxCount: maxCount,
		order:    list.New(),
		items:    make(map[K]lruEntry[K, V]),
	}
}

func (m *LRUMap[K, V]) Put(key K, value V) (bool, K, V) {
	if entry, ok := m.items[key]; ok {
		entry.value = value
		m.items[key] = entry
		m.order.MoveToBack(entry.node)
		var zeroK K
		var zeroV V
		return false, zeroK, zeroV
	}

	node := m.order.PushBack(key)
	m.items[key] = lruEntry[K, V]{value: value, node: node}

	if len(m.items) <= m.maxCount {
		var zeroK K
		var zeroV V
		return false, zeroK, zeroV
	}

	front := m.order.Front()
	if front == nil {
		var zeroK K
		var zeroV V
		return false, zeroK, zeroV
	}

	evictedKey, _ := front.Value.(K)
	evictedVal, _ := m.Remove(evictedKey)
	return true, evictedKey, evictedVal
}

func (m *LRUMap[K, V]) Get(key K) (V, bool) {
	entry, ok := m.items[key]
	if !ok {
		var zero V
		return zero, false
	}
	m.order.MoveToBack(entry.node)
	return entry.value, true
}

func (m *LRUMap[K, V]) Take(key K) (V, bool) {
	return m.Remove(key)
}

func (m *LRUMap[K, V]) Remove(key K) (V, bool) {
	entry, ok := m.items[key]
	if !ok {
		var zero V
		return zero, false
	}

	m.order.Remove(entry.node)
	delete(m.items, key)
	return entry.value, true
}

func (m *LRUMap[K, V]) Len() int {
	return len(m.items)
}
