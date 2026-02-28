package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type DiskCache struct {
	mu          sync.Mutex
	dir         string
	maxBytes    int64
	currentSize int64
	index       map[string]*diskMeta
}

type diskMeta struct {
	Key          string              `json:"key"`
	Header       map[string][]string `json:"header"`
	StatusCode   int                 `json:"status_code"`
	CreatedAt    time.Time           `json:"created_at"`
	ExpiresAt    time.Time           `json:"expires_at"`
	ETag         string              `json:"etag"`
	LastModified string              `json:"last_modified"`
	SizeBytes    int64               `json:"size_bytes"`
	LastAccessed time.Time           `json:"last_accessed"`
}

func NewDiskCache(dir string, maxBytes int64) (*DiskCache, error) {
	if dir == "" {
		return nil, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &DiskCache{
		dir:      dir,
		maxBytes: maxBytes,
		index:    make(map[string]*diskMeta),
	}, nil
}

func (d *DiskCache) Get(key string) (*CacheEntry, bool) {
	if d == nil {
		return nil, false
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	meta := d.index[key]
	if meta == nil {
		metaPath := d.metaPath(key)
		rawMeta, err := os.ReadFile(metaPath)
		if err != nil {
			return nil, false
		}
		meta = &diskMeta{}
		if err := json.Unmarshal(rawMeta, meta); err != nil {
			return nil, false
		}
		d.index[key] = meta
	}

	if meta.ExpiresAt.Before(time.Now()) {
		d.removeLocked(key)
		return nil, false
	}

	body, err := os.ReadFile(d.bodyPath(key))
	if err != nil {
		return nil, false
	}
	meta.LastAccessed = time.Now()
	_ = d.writeMetaLocked(key, meta)

	return &CacheEntry{
		data:         body,
		header:       http.Header(meta.Header),
		statusCode:   meta.StatusCode,
		createdAt:    meta.CreatedAt,
		expiresAt:    meta.ExpiresAt,
		eTag:         meta.ETag,
		lastModified: meta.LastModified,
		sizeBytes:    meta.SizeBytes,
	}, true
}

func (d *DiskCache) Set(key string, entry *CacheEntry) {
	if d == nil || entry == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	meta := &diskMeta{
		Key:          key,
		Header:       map[string][]string(entry.header.Clone()),
		StatusCode:   entry.statusCode,
		CreatedAt:    entry.createdAt,
		ExpiresAt:    entry.expiresAt,
		ETag:         entry.eTag,
		LastModified: entry.lastModified,
		SizeBytes:    int64(len(entry.data)),
		LastAccessed: time.Now(),
	}

	if err := os.WriteFile(d.bodyPath(key), entry.data, 0o644); err != nil {
		return
	}
	if err := d.writeMetaLocked(key, meta); err != nil {
		return
	}

	if old := d.index[key]; old != nil {
		d.currentSize -= old.SizeBytes
	}
	d.index[key] = meta
	d.currentSize += meta.SizeBytes
	d.evictIfNeededLocked()
}

func (d *DiskCache) writeMetaLocked(key string, meta *diskMeta) error {
	raw, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(d.metaPath(key), raw, 0o644)
}

func (d *DiskCache) evictIfNeededLocked() {
	if d.maxBytes <= 0 {
		return
	}
	for d.currentSize > d.maxBytes {
		var items []*diskMeta
		for _, m := range d.index {
			items = append(items, m)
		}
		if len(items) == 0 {
			return
		}
		sort.Slice(items, func(i, j int) bool {
			return items[i].LastAccessed.Before(items[j].LastAccessed)
		})
		d.removeLocked(items[0].Key)
	}
}

func (d *DiskCache) removeLocked(key string) {
	if meta := d.index[key]; meta != nil {
		d.currentSize -= meta.SizeBytes
	}
	delete(d.index, key)
	_ = os.Remove(d.bodyPath(key))
	_ = os.Remove(d.metaPath(key))
}

func (d *DiskCache) bodyPath(key string) string {
	return filepath.Join(d.dir, safeKey(key)+".body")
}

func (d *DiskCache) metaPath(key string) string {
	return filepath.Join(d.dir, safeKey(key)+".meta")
}

func safeKey(key string) string {
	sum := sha1.Sum([]byte(strings.TrimSpace(key)))
	return hex.EncodeToString(sum[:])
}
