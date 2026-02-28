package main

import (
	"container/list"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CacheEntry struct {
	data         []byte
	header       http.Header
	statusCode   int
	createdAt    time.Time
	expiresAt    time.Time
	eTag         string
	lastModified string
	sizeBytes    int64
}

type Cache struct {
	mu          sync.RWMutex
	store       map[string]*list.Element
	lru         *list.List
	ttl         time.Duration
	maxBytes    int64
	currentSize int64
	varyByBase  map[string][]string
	policy      EvictionPolicy
}

type EvictionPolicy string

const (
	EvictionLRU EvictionPolicy = "lru"
	EvictionLFU EvictionPolicy = "lfu"
)

type cacheItem struct {
	key        string
	entry      *CacheEntry
	hits       uint64
	lastAccess time.Time
}

func NewCache(ttl time.Duration, maxBytes int64) *Cache {
	return &Cache{
		store:      make(map[string]*list.Element),
		lru:        list.New(),
		ttl:        ttl,
		maxBytes:   maxBytes,
		varyByBase: make(map[string][]string),
		policy:     EvictionLRU,
	}
}

func (c *Cache) SetEvictionPolicy(policy EvictionPolicy) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if policy != EvictionLFU {
		c.policy = EvictionLRU
		return
	}
	c.policy = EvictionLFU
}

func (c *Cache) Get(key string) (*CacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.store[key]
	if !ok {
		return nil, false
	}
	item := elem.Value.(*cacheItem)
	if item.entry.expiresAt.Before(time.Now()) {
		c.removeElement(elem)
		return nil, false
	}

	item.hits++
	item.lastAccess = time.Now()
	c.lru.MoveToFront(elem)
	return cloneEntry(item.entry), true
}

func (c *Cache) GetStale(key string) (*CacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.store[key]
	if !ok {
		return nil, false
	}

	item := elem.Value.(*cacheItem)
	item.hits++
	item.lastAccess = time.Now()
	c.lru.MoveToFront(elem)
	return cloneEntry(item.entry), true
}

func (c *Cache) Set(key string, entry *CacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry.sizeBytes = estimateEntrySize(key, entry)
	if elem, ok := c.store[key]; ok {
		existing := elem.Value.(*cacheItem)
		c.currentSize -= existing.entry.sizeBytes
		existing.entry = cloneEntry(entry)
		existing.hits++
		existing.lastAccess = time.Now()
		c.currentSize += existing.entry.sizeBytes
		c.lru.MoveToFront(elem)
	} else {
		item := &cacheItem{
			key:        key,
			entry:      cloneEntry(entry),
			hits:       1,
			lastAccess: time.Now(),
		}
		elem := c.lru.PushFront(item)
		c.store[key] = elem
		c.currentSize += item.entry.sizeBytes
	}

	c.evictIfNeeded()
}

func (c *Cache) SetWithTTL(key string, data []byte, header http.Header, statusCode int, ttl time.Duration) {
	entry := &CacheEntry{
		data:         data,
		header:       header.Clone(),
		statusCode:   statusCode,
		createdAt:    time.Now(),
		expiresAt:    time.Now().Add(ttl),
		eTag:         header.Get("ETag"),
		lastModified: header.Get("Last-Modified"),
	}
	c.Set(key, entry)
}

func cacheKey(r *http.Request) string {
	return r.URL.String()
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.store[key]; ok {
		c.removeElement(elem)
	}
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store = make(map[string]*list.Element)
	c.lru.Init()
	c.currentSize = 0
	c.varyByBase = make(map[string][]string)
}

func (c *Cache) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store = make(map[string]*list.Element)
	c.lru.Init()
	c.currentSize = 0
	c.varyByBase = make(map[string][]string)
}

func (c *Cache) StartCleanup() {
	go func() {
		for {
			time.Sleep(c.ttl)
			c.mu.Lock()
			for _, elem := range c.store {
				item := elem.Value.(*cacheItem)
				if item.entry.expiresAt.Before(time.Now()) {
					c.removeElement(elem)
				}
			}
			c.mu.Unlock()
		}
	}()
}

func (c *Cache) StopCleanup() {
	c.Clear()
}

func (c *Cache) LookupKey(baseKey string, r *http.Request) string {
	c.mu.RLock()
	varyHeaders := append([]string(nil), c.varyByBase[baseKey]...)
	c.mu.RUnlock()
	return buildCacheKey(baseKey, r, varyHeaders)
}

func (c *Cache) UpdateVary(baseKey string, responseHeader http.Header) bool {
	headers, cacheable := parseVaryHeaders(responseHeader.Values("Vary"))
	if !cacheable {
		return false
	}

	c.mu.Lock()
	c.varyByBase[baseKey] = headers
	c.mu.Unlock()
	return true
}

func (c *Cache) removeElement(elem *list.Element) {
	item := elem.Value.(*cacheItem)
	c.currentSize -= item.entry.sizeBytes
	delete(c.store, item.key)
	c.lru.Remove(elem)
}

func (c *Cache) evictIfNeeded() {
	if c.maxBytes <= 0 {
		return
	}
	for c.currentSize > c.maxBytes {
		victim := c.selectVictim()
		if victim == nil {
			return
		}
		c.removeElement(victim)
	}
}

func (c *Cache) selectVictim() *list.Element {
	if c.policy == EvictionLFU {
		var (
			victimElem *list.Element
			minHits    uint64
			minAccess  time.Time
		)
		for _, elem := range c.store {
			item := elem.Value.(*cacheItem)
			if victimElem == nil ||
				item.hits < minHits ||
				(item.hits == minHits && item.lastAccess.Before(minAccess)) {
				victimElem = elem
				minHits = item.hits
				minAccess = item.lastAccess
			}
		}
		return victimElem
	}
	return c.lru.Back()
}

func cloneEntry(entry *CacheEntry) *CacheEntry {
	if entry == nil {
		return nil
	}
	cloned := *entry
	cloned.header = entry.header.Clone()
	cloned.data = append([]byte(nil), entry.data...)
	return &cloned
}

func estimateEntrySize(key string, entry *CacheEntry) int64 {
	size := int64(len(key) + len(entry.data))
	for hk, values := range entry.header {
		size += int64(len(hk))
		for _, v := range values {
			size += int64(len(v))
		}
	}
	return size
}

func cacheBaseKey(r *http.Request) string {
	return r.Method + ":" + r.URL.String()
}

func buildCacheKey(baseKey string, r *http.Request, varyHeaders []string) string {
	if len(varyHeaders) == 0 {
		return baseKey
	}
	var b strings.Builder
	b.WriteString(baseKey)
	for _, h := range varyHeaders {
		b.WriteString("|")
		b.WriteString(h)
		b.WriteString("=")
		b.WriteString(strings.TrimSpace(r.Header.Get(h)))
	}
	return b.String()
}

func parseVaryHeaders(varyValues []string) ([]string, bool) {
	seen := make(map[string]bool)
	var headers []string

	for _, varyValue := range varyValues {
		for _, part := range strings.Split(varyValue, ",") {
			h := strings.ToLower(strings.TrimSpace(part))
			if h == "" {
				continue
			}
			if h == "*" {
				return nil, false
			}
			if !seen[h] {
				seen[h] = true
				headers = append(headers, h)
			}
		}
	}
	return headers, true
}

// get ttl

func getTTL(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}

	flags, maxAges := parseCacheControl(resp.Header.Values("Cache-Control"))

	// This cache does not support validation (ETag / If-None-Match) yet,
	// so treat revalidation-required responses as uncacheable.
	if flags["no-store"] || flags["no-cache"] {
		return 0
	}

	freshnessSeconds := int64(-1)
	if v, ok := maxAges["s-maxage"]; ok {
		freshnessSeconds = v
	} else if v, ok := maxAges["max-age"]; ok {
		freshnessSeconds = v
	} else if expires := resp.Header.Get("Expires"); expires != "" {
		if expTime, err := http.ParseTime(expires); err == nil {
			base := time.Now()
			if date := resp.Header.Get("Date"); date != "" {
				if dateTime, err := http.ParseTime(date); err == nil {
					base = dateTime
				}
			}
			freshnessSeconds = int64(expTime.Sub(base) / time.Second)
		}
	}

	if freshnessSeconds <= 0 {
		return 0
	}

	ageSeconds := parseAgeHeader(resp.Header.Get("Age"))
	remaining := freshnessSeconds - ageSeconds
	if remaining <= 0 {
		return 0
	}

	return time.Duration(remaining) * time.Second
}

func parseCacheControl(headers []string) (map[string]bool, map[string]int64) {
	flags := make(map[string]bool)
	maxAges := make(map[string]int64)

	for _, headerValue := range headers {
		for _, directive := range splitCacheControlDirectives(headerValue) {
			directive = strings.TrimSpace(directive)
			if directive == "" {
				continue
			}

			name, value, hasValue := strings.Cut(directive, "=")
			name = strings.ToLower(strings.TrimSpace(name))
			if name == "" {
				continue
			}

			if !hasValue {
				flags[name] = true
				continue
			}

			value = strings.TrimSpace(value)
			if unquoted, err := strconv.Unquote(value); err == nil {
				value = unquoted
			}

			if name == "max-age" || name == "s-maxage" {
				seconds, err := strconv.ParseInt(value, 10, 64)
				if err != nil || seconds < 0 {
					continue
				}
				// Be conservative for duplicate directives: keep the shortest lifetime.
				if current, ok := maxAges[name]; !ok || seconds < current {
					maxAges[name] = seconds
				}
			}

			if name == "no-cache" || name == "no-store" {
				flags[name] = true
			}
		}
	}

	return flags, maxAges
}

func splitCacheControlDirectives(s string) []string {
	var directives []string
	start := 0
	inQuotes := false
	escaped := false

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inQuotes {
			escaped = true
			continue
		}
		if ch == '"' {
			inQuotes = !inQuotes
			continue
		}
		if ch == ',' && !inQuotes {
			directives = append(directives, s[start:i])
			start = i + 1
		}
	}

	directives = append(directives, s[start:])
	return directives
}

func parseAgeHeader(ageValue string) int64 {
	ageValue = strings.TrimSpace(ageValue)
	if ageValue == "" {
		return 0
	}

	age, err := strconv.ParseInt(ageValue, 10, 64)
	if err != nil || age < 0 {
		return 0
	}
	return age
}

func main() {
	cfg := loadEdgeConfigFromEnv()

	cache := NewCache(10*time.Minute, cfg.MaxMemoryBytes)
	cache.SetEvictionPolicy(EvictionPolicy(strings.ToLower(getEnv("EDGE_EVICTION_POLICY", "lru"))))

	disk, err := NewDiskCache(cfg.DiskCacheDir, cfg.DiskCacheMaxBytes)
	if err != nil {
		log.Fatalf("failed to initialize disk cache: %v", err)
	}

	origins := cfg.Origins
	if len(origins) == 0 {
		origins = []string{cfg.OriginURL}
	}

	var ring *HashRing
	if len(origins) > 1 {
		ring = NewHashRing(origins, cfg.HashReplicas)
	}

	proxy := &EdgeServer{
		origin:  cfg.OriginURL,
		origins: origins,
		shield:  cfg.ShieldURL,
		cache:   cache,
		disk:    disk,
		client:  newUpstreamClient(cfg),
		ring:    ring,
	}
	cache.StartCleanup()
	http.HandleFunc("/", proxy.ServeHTTP)
	log.Printf("Edge server listening on %s (origins=%v shield=%q)", cfg.ListenAddr, origins, cfg.ShieldURL)

	server := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      nil,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		log.Fatal(server.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile))
		return
	}

	// Keep optional cert env visible to operators.
	if os.Getenv("EDGE_TLS_CERT_FILE") != "" || os.Getenv("EDGE_TLS_KEY_FILE") != "" {
		log.Println("EDGE_TLS_CERT_FILE/EDGE_TLS_KEY_FILE must both be set to enable TLS")
	}

	log.Fatal(server.ListenAndServe())
}
