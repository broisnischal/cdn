package main

import (
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"
)

type EdgeServer struct {
	origin   string
	origins  []string
	shield   string
	cache    *Cache
	disk     *DiskCache
	client   *http.Client
	ring     *HashRing
	inflight singleflight.Group
}

type originResult struct {
	header      http.Header
	statusCode  int
	body        []byte
	cacheStatus string
}

func (es *EdgeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		es.serveNoCache(w, r)
		return
	}

	baseKey := cacheBaseKey(r)
	key := es.cache.LookupKey(baseKey, r)

	if entry, found := es.cache.Get(key); found {
		serveCachedEntry(w, r, entry, "HIT")
		return
	}

	if entry, found := es.disk.Get(key); found {
		es.cache.Set(key, entry)
		serveCachedEntry(w, r, entry, "HIT-DISK")
		return
	}

	if r.Header.Get("Range") != "" {
		es.serveNoCache(w, r)
		return
	}

	result, err, _ := es.inflight.Do(key, func() (interface{}, error) {
		if entry, found := es.cache.Get(key); found {
			return &originResult{
				header:      entry.header.Clone(),
				statusCode:  entry.statusCode,
				body:        append([]byte(nil), entry.data...),
				cacheStatus: "HIT",
			}, nil
		}

		staleEntry, hasStale := es.cache.GetStale(key)
		return es.fetchFromOrigin(r, baseKey, key, staleEntry, hasStale)
	})
	if err != nil {
		http.Error(w, "Origin fetch failed", http.StatusBadGateway)
		return
	}

	final := result.(*originResult)
	copyHeaders(w.Header(), final.header)
	w.Header().Set("X-Cache", final.cacheStatus)
	writeResponseWithRange(w, r, final.statusCode, final.header, final.body)
}

func serveCachedEntry(w http.ResponseWriter, r *http.Request, entry *CacheEntry, cacheStatus string) {
	copyHeaders(w.Header(), entry.header)
	w.Header().Set("X-Cache", cacheStatus)
	writeResponseWithRange(w, r, entry.statusCode, entry.header, entry.data)
}

func (es *EdgeServer) fetchFromOrigin(r *http.Request, baseKey, fallbackKey string, staleEntry *CacheEntry, hasStale bool) (*originResult, error) {
	upstream := es.chooseUpstream(fallbackKey)
	originURL := strings.TrimRight(upstream, "/") + r.URL.Path
	if r.URL.RawQuery != "" {
		originURL += "?" + r.URL.RawQuery
	}
	req, _ := http.NewRequestWithContext(r.Context(), r.Method, originURL, nil)
	copyHeaders(req.Header, r.Header)

	if hasStale {
		if staleEntry.eTag != "" {
			req.Header.Set("If-None-Match", staleEntry.eTag)
		}
		if staleEntry.lastModified != "" {
			req.Header.Set("If-Modified-Since", staleEntry.lastModified)
		}
	}

	resp, err := es.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified && hasStale {
		newTTL := getTTL(resp)
		if newTTL > 0 {
			staleEntry.expiresAt = time.Now().Add(newTTL)
			if etag := resp.Header.Get("ETag"); etag != "" {
				staleEntry.eTag = etag
			}
			if lm := resp.Header.Get("Last-Modified"); lm != "" {
				staleEntry.lastModified = lm
			}
			es.cache.Set(fallbackKey, staleEntry)
		}
		return &originResult{
			header:      staleEntry.header.Clone(),
			statusCode:  staleEntry.statusCode,
			body:        append([]byte(nil), staleEntry.data...),
			cacheStatus: "REVALIDATED",
		}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ttl := getTTL(resp)
	cacheStatus := "BYPASS"
	if ttl > 0 {
		if es.cache.UpdateVary(baseKey, resp.Header) {
			storeKey := es.cache.LookupKey(baseKey, r)
			entry := &CacheEntry{
				data:         body,
				header:       resp.Header.Clone(),
				statusCode:   resp.StatusCode,
				createdAt:    time.Now(),
				expiresAt:    time.Now().Add(ttl),
				eTag:         resp.Header.Get("ETag"),
				lastModified: resp.Header.Get("Last-Modified"),
			}
			es.cache.Set(storeKey, entry)
			es.disk.Set(storeKey, entry)
			cacheStatus = "MISS"
		}
	}

	return &originResult{
		header:      resp.Header.Clone(),
		statusCode:  resp.StatusCode,
		body:        body,
		cacheStatus: cacheStatus,
	}, nil
}

func (es *EdgeServer) serveNoCache(w http.ResponseWriter, r *http.Request) {
	baseKey := cacheBaseKey(r)
	upstream := es.chooseUpstream(baseKey)
	originURL := strings.TrimRight(upstream, "/") + r.URL.Path
	if r.URL.RawQuery != "" {
		originURL += "?" + r.URL.RawQuery
	}

	req, _ := http.NewRequestWithContext(r.Context(), r.Method, originURL, r.Body)
	copyHeaders(req.Header, r.Header)

	resp, err := es.client.Do(req)
	if err != nil {
		http.Error(w, "Origin fetch failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read origin response", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("Range") != "" {
		w.Header().Set("X-Cache", "MISS-RANGE")
	} else {
		w.Header().Set("X-Cache", "BYPASS")
	}
	copyHeaders(w.Header(), resp.Header)
	writeResponseWithRange(w, r, resp.StatusCode, resp.Header, body)
}

func (es *EdgeServer) chooseUpstream(cacheKey string) string {
	if es.shield != "" {
		return es.shield
	}
	if es.ring != nil {
		if node := es.ring.GetNode(cacheKey); node != "" {
			return node
		}
	}
	if len(es.origins) > 0 {
		return es.origins[0]
	}
	return es.origin
}

func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
