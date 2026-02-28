package main

import (
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
)

type HashRing struct {
	mu        sync.RWMutex
	replicas  int
	ring      []uint32
	nodeByKey map[uint32]string
	nodes     []string
}

func NewHashRing(nodes []string, replicas int) *HashRing {
	if replicas <= 0 {
		replicas = 100
	}
	hr := &HashRing{
		replicas:  replicas,
		nodeByKey: make(map[uint32]string),
	}
	hr.SetNodes(nodes)
	return hr
}

func (h *HashRing) SetNodes(nodes []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.ring = h.ring[:0]
	h.nodeByKey = make(map[uint32]string)
	h.nodes = append([]string(nil), nodes...)

	for _, node := range nodes {
		for i := 0; i < h.replicas; i++ {
			key := crc32.ChecksumIEEE([]byte(node + "#" + strconv.Itoa(i)))
			h.ring = append(h.ring, key)
			h.nodeByKey[key] = node
		}
	}

	sort.Slice(h.ring, func(i, j int) bool {
		return h.ring[i] < h.ring[j]
	})
}

func (h *HashRing) GetNode(key string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.ring) == 0 {
		return ""
	}

	hash := crc32.ChecksumIEEE([]byte(key))
	idx := sort.Search(len(h.ring), func(i int) bool {
		return h.ring[i] >= hash
	})
	if idx == len(h.ring) {
		idx = 0
	}
	return h.nodeByKey[h.ring[idx]]
}
