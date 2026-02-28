package main

import (
	"log"
	"math/rand"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
)

type geoRule struct {
	prefix netip.Prefix
	pool   string
}

type weightedA struct {
	ip     net.IP
	weight int
}

type dnsRouter struct {
	domain      string
	defaultPool string
	rules       []geoRule
	pools       map[string][]weightedA
}

func main() {
	rand.Seed(time.Now().UnixNano())

	router := loadRouterFromEnv()
	mux := dns.NewServeMux()
	mux.HandleFunc(".", router.handleQuery)

	addr := getEnv("DNS_LISTEN_ADDR", ":5353")
	server := &dns.Server{Addr: addr, Net: "udp", Handler: mux}
	log.Printf("Geo DNS listening on %s for domain %s", addr, router.domain)
	log.Fatal(server.ListenAndServe())
}

func (r *dnsRouter) handleQuery(w dns.ResponseWriter, req *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(req)
	msg.Authoritative = true

	clientIP := parseClientIP(w.RemoteAddr().String())

	for _, q := range req.Question {
		if q.Qtype != dns.TypeA {
			continue
		}
		if !dns.IsSubDomain(r.domain, strings.ToLower(q.Name)) {
			continue
		}
		pool := r.poolForIP(clientIP)
		targets := r.pools[pool]
		if len(targets) == 0 {
			continue
		}
		selected := pickWeighted(targets)
		if selected == nil {
			continue
		}
		rr := &dns.A{
			Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 20},
			A:   selected,
		}
		msg.Answer = append(msg.Answer, rr)
	}

	_ = w.WriteMsg(msg)
}

func (r *dnsRouter) poolForIP(ip netip.Addr) string {
	if ip.IsValid() {
		for _, rule := range r.rules {
			if rule.prefix.Contains(ip) {
				return rule.pool
			}
		}
	}
	return r.defaultPool
}

func pickWeighted(items []weightedA) net.IP {
	total := 0
	for _, it := range items {
		total += it.weight
	}
	if total <= 0 {
		return nil
	}
	n := rand.Intn(total)
	acc := 0
	for _, it := range items {
		acc += it.weight
		if n < acc {
			return it.ip
		}
	}
	return items[len(items)-1].ip
}

func loadRouterFromEnv() *dnsRouter {
	domain := strings.ToLower(getEnv("DNS_DOMAIN", "cdn.local."))
	if !strings.HasSuffix(domain, ".") {
		domain += "."
	}
	defaultPool := getEnv("DNS_DEFAULT_POOL", "default")

	pools := map[string][]weightedA{}
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		if !strings.HasPrefix(key, "DNS_POOL_") {
			continue
		}
		pool := strings.ToLower(strings.TrimPrefix(key, "DNS_POOL_"))
		pools[pool] = parseWeightedPool(parts[1])
	}
	if len(pools[defaultPool]) == 0 {
		pools[defaultPool] = parseWeightedPool(getEnv("DNS_POOL_DEFAULT", "127.0.0.1:100"))
	}

	var rules []geoRule
	for _, part := range strings.Split(getEnv("DNS_GEO_RULES", ""), ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		pair := strings.SplitN(part, "=", 2)
		if len(pair) != 2 {
			continue
		}
		prefix, err := netip.ParsePrefix(strings.TrimSpace(pair[0]))
		if err != nil {
			continue
		}
		pool := strings.ToLower(strings.TrimSpace(pair[1]))
		rules = append(rules, geoRule{prefix: prefix, pool: pool})
	}

	return &dnsRouter{
		domain:      domain,
		defaultPool: defaultPool,
		rules:       rules,
		pools:       pools,
	}
}

func parseWeightedPool(raw string) []weightedA {
	var out []weightedA
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		pair := strings.SplitN(part, ":", 2)
		if len(pair) != 2 {
			continue
		}
		ip := net.ParseIP(strings.TrimSpace(pair[0]))
		if ip == nil {
			continue
		}
		weight, err := strconv.Atoi(strings.TrimSpace(pair[1]))
		if err != nil || weight <= 0 {
			continue
		}
		out = append(out, weightedA{ip: ip, weight: weight})
	}
	return out
}

func parseClientIP(remote string) netip.Addr {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		return netip.Addr{}
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}
	return ip
}

func getEnv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
