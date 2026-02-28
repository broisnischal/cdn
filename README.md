# Go CDN Stack

This repo now includes a mini CDN platform:

- `origin/` static origin server (`index.html` + high-res image)
- `cdn/` edge/shield server with memory+disk cache, vary support, and request collapsing
- `dns/` geo-aware weighted DNS server
- `compose.yaml` full local stack
- `terraform/` Docker-provider deployment

## Quick Start (Docker Compose)

```bash
docker compose -f compose.yaml up -d --build
curl -i http://localhost:8080/
curl -i http://localhost:8080/images/hero-4k.svg
curl -i -H "Range: bytes=0-1023" http://localhost:8080/images/hero-4k.svg
```

Stop:

```bash
docker compose -f compose.yaml down -v
```

## Runtime Config

Use `.env` (or `.env.example`) for runtime variables like:

- edge/shield origins
- cache limits and eviction policy
- disk cache path/size
- TLS cert/key
- DNS domain, pools, and CIDR rules

## Local Run (without Docker)

```bash
cd origin && go run .
cd cdn && go run .
cd dns && go run .
```

## Just Commands

```bash
just up
just logs edge
just down
just terraform-init
just terraform-apply
```
