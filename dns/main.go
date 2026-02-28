package main

import (
	"fmt"
	"log"
	"math"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"

	"github.com/miekg/dns"
	"github.com/oschwald/geoip2-golang"
)

type cidrRule struct {
	prefix   netip.Prefix
	edgeName string
}

type edgeServer struct {
	name string
	ip   net.IP
	lat  float64
	lon  float64
}

type dnsRouter struct {
	authoritativeDomain string
	originIP            net.IP
	defaultEdge         string
	cidrRules           []cidrRule
	edgesByName         map[string]edgeServer
	geoDB               *geoip2.Reader
}

func main() {
	router, err := loadRouterFromEnv()
	if err != nil {
		log.Fatalf("failed to load DNS router config: %v", err)
	}
	defer func() {
		if router.geoDB != nil {
			_ = router.geoDB.Close()
		}
	}()
	mux := dns.NewServeMux()
	mux.HandleFunc(".", router.handleQuery)

	addr := getEnv("DNS_LISTEN_ADDR", ":5353")
	udp := &dns.Server{Addr: addr, Net: "udp", Handler: mux}
	tcp := &dns.Server{Addr: addr, Net: "tcp", Handler: mux}
	log.Printf("Authoritative DNS listening on %s for domain %s", addr, router.authoritativeDomain)

	go func() {
		if err := tcp.ListenAndServe(); err != nil {
			log.Fatalf("tcp dns failed: %v", err)
		}
	}()
	log.Fatal(udp.ListenAndServe())
}

func (r *dnsRouter) handleQuery(w dns.ResponseWriter, req *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(req)
	msg.Authoritative = true

	clientIP := parseClientAddr(w.RemoteAddr().String())

	for _, q := range req.Question {
		domainName := strings.ToLower(q.Name)
		if !dns.IsSubDomain(r.authoritativeDomain, domainName) {
			msg.Rcode = dns.RcodeRefused
			continue
		}

		if q.Qtype != dns.TypeA && q.Qtype != dns.TypeANY {
			continue
		}

		if domainName == "origin."+r.authoritativeDomain {
			rr := &dns.A{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 50},
				A:   r.originIP,
			}
			msg.Answer = append(msg.Answer, rr)
			continue
		}

		edgeIP, ok := r.pickEdgeIP(clientIP)
		if !ok {
			msg.Rcode = dns.RcodeServerFailure
			continue
		}

		rr := &dns.A{
			Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 20},
			A:   edgeIP,
		}
		msg.Answer = append(msg.Answer, rr)
	}

	_ = w.WriteMsg(msg)
}

func (r *dnsRouter) pickEdgeIP(clientAddr netip.Addr) (net.IP, bool) {
	if clientAddr.IsValid() {
		for _, rule := range r.cidrRules {
			if rule.prefix.Contains(clientAddr) {
				if edge, ok := r.edgesByName[rule.edgeName]; ok {
					return edge.ip, true
				}
			}
		}
	}

	if clientAddr.IsValid() && r.geoDB != nil {
		clientIP := net.ParseIP(clientAddr.String())
		if clientIP != nil {
			if lat, lon, ok := lookupGeoCoords(clientIP, r.geoDB); ok {
				if edgeIP, ok := nearestEdgeIP(lat, lon, r.edgesByName); ok {
					return edgeIP, true
				}
			}
		}
	}

	if edge, ok := r.edgesByName[r.defaultEdge]; ok {
		return edge.ip, true
	}

	for _, edge := range r.edgesByName {
		return edge.ip, true
	}
	return nil, false
}

func nearestEdgeIP(clientLat, clientLon float64, edges map[string]edgeServer) (net.IP, bool) {
	var (
		bestIP       net.IP
		bestDistance = math.Inf(1)
	)
	for _, edge := range edges {
		d := haversineKM(clientLat, clientLon, edge.lat, edge.lon)
		if d < bestDistance {
			bestDistance = d
			bestIP = edge.ip
		}
	}
	if bestIP == nil {
		return nil, false
	}
	return bestIP, true
}

func loadRouterFromEnv() (*dnsRouter, error) {
	authoritative := strings.ToLower(getEnv("DNS_AUTHORITATIVE_DOMAIN", "cdn.local."))
	if !strings.HasSuffix(authoritative, ".") {
		authoritative += "."
	}

	originIP := net.ParseIP(strings.TrimSpace(os.Getenv("DNS_ORIGIN_IP")))
	if originIP == nil {
		return nil, fmt.Errorf("DNS_ORIGIN_IP is required and must be valid")
	}

	edgesByName, err := parseEdgeServers(getEnv("DNS_EDGE_SERVERS", "us|127.0.0.1|37.7749|-122.4194"))
	if err != nil {
		return nil, err
	}

	defaultEdge := strings.ToLower(getEnv("DNS_DEFAULT_EDGE", "us"))
	if _, ok := edgesByName[defaultEdge]; !ok {
		for name := range edgesByName {
			defaultEdge = name
			break
		}
	}

	var rules []cidrRule
	for _, part := range strings.Split(getEnv("DNS_GEO_CIDR_RULES", ""), ",") {
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
		edgeName := strings.ToLower(strings.TrimSpace(pair[1]))
		if _, ok := edgesByName[edgeName]; !ok {
			continue
		}
		rules = append(rules, cidrRule{prefix: prefix, edgeName: edgeName})
	}

	var geoDB *geoip2.Reader
	if dbPath := strings.TrimSpace(os.Getenv("DNS_GEOIP_DB_PATH")); dbPath != "" {
		reader, err := geoip2.Open(dbPath)
		if err != nil {
			log.Printf("geoip db not loaded: %v", err)
		} else {
			geoDB = reader
			log.Printf("geoip db loaded from %s", dbPath)
		}
	}

	return &dnsRouter{
		authoritativeDomain: authoritative,
		originIP:            originIP,
		defaultEdge:         defaultEdge,
		cidrRules:           rules,
		edgesByName:         edgesByName,
		geoDB:               geoDB,
	}, nil
}

func parseEdgeServers(raw string) (map[string]edgeServer, error) {
	out := make(map[string]edgeServer)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		fields := strings.Split(part, "|")
		if len(fields) != 4 {
			return nil, fmt.Errorf("invalid DNS_EDGE_SERVERS entry: %q", part)
		}
		name := strings.ToLower(strings.TrimSpace(fields[0]))
		ip := net.ParseIP(strings.TrimSpace(fields[1]))
		if ip == nil {
			return nil, fmt.Errorf("invalid edge ip in DNS_EDGE_SERVERS: %q", fields[1])
		}
		lat, err := parseFloat(strings.TrimSpace(fields[2]))
		if err != nil {
			return nil, fmt.Errorf("invalid latitude in DNS_EDGE_SERVERS: %w", err)
		}
		lon, err := parseFloat(strings.TrimSpace(fields[3]))
		if err != nil {
			return nil, fmt.Errorf("invalid longitude in DNS_EDGE_SERVERS: %w", err)
		}
		out[name] = edgeServer{
			name: name,
			ip:   ip,
			lat:  lat,
			lon:  lon,
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("DNS_EDGE_SERVERS must contain at least one edge")
	}
	return out, nil
}

func parseClientAddr(remote string) netip.Addr {
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

func lookupGeoCoords(ip net.IP, reader *geoip2.Reader) (float64, float64, bool) {
	record, err := reader.City(ip)
	if err != nil {
		return 0, 0, false
	}
	if record.Location.Latitude == 0 && record.Location.Longitude == 0 {
		return 0, 0, false
	}
	return record.Location.Latitude, record.Location.Longitude, true
}

func haversineKM(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKM = 6371.0
	dLat := toRadians(lat2 - lat1)
	dLon := toRadians(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRadians(lat1))*math.Cos(toRadians(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKM * c
}

func toRadians(v float64) float64 {
	return v * math.Pi / 180
}

func parseFloat(v string) (float64, error) {
	return strconv.ParseFloat(v, 64)
}

func getEnv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
