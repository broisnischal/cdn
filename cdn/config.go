package main

import (
	"crypto/tls"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type EdgeConfig struct {
	ListenAddr         string
	OriginURL          string
	Origins            []string
	ShieldURL          string
	HashReplicas       int
	MaxMemoryBytes     int64
	DiskCacheDir       string
	DiskCacheMaxBytes  int64
	ClientTimeout      time.Duration
	InsecureUpstreamTL bool
	TLSCertFile        string
	TLSKeyFile         string
}

func loadEdgeConfigFromEnv() EdgeConfig {
	return EdgeConfig{
		ListenAddr:         getEnv("EDGE_LISTEN_ADDR", ":8080"),
		OriginURL:          getEnv("ORIGIN_URL", "http://localhost:8081"),
		Origins:            splitCSV(os.Getenv("ORIGINS")),
		ShieldURL:          strings.TrimSpace(os.Getenv("SHIELD_URL")),
		HashReplicas:       getEnvInt("HASH_REPLICAS", 100),
		MaxMemoryBytes:     getEnvInt64("EDGE_MAX_MEMORY_BYTES", 128*1024*1024),
		DiskCacheDir:       strings.TrimSpace(os.Getenv("EDGE_DISK_CACHE_DIR")),
		DiskCacheMaxBytes:  getEnvInt64("EDGE_DISK_CACHE_MAX_BYTES", 2*1024*1024*1024),
		ClientTimeout:      time.Duration(getEnvInt("UPSTREAM_TIMEOUT_SEC", 10)) * time.Second,
		InsecureUpstreamTL: getEnvBool("UPSTREAM_INSECURE_TLS", false),
		TLSCertFile:        strings.TrimSpace(os.Getenv("EDGE_TLS_CERT_FILE")),
		TLSKeyFile:         strings.TrimSpace(os.Getenv("EDGE_TLS_KEY_FILE")),
	}
}

func newUpstreamClient(cfg EdgeConfig) *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: cfg.InsecureUpstreamTL,
		},
	}
	return &http.Client{
		Timeout:   cfg.ClientTimeout,
		Transport: tr,
	}
}

func getEnv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func getEnvInt64(key string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return v
}

func getEnvBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var out []string
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
