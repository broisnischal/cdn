project_name        = "gocdn-dev"
authoritative_domain = "cdn-dev.example.com."
edge_instance_type  = "t3.small"
origin_instance_type = "t3.small"
dns_instance_type    = "t3.small"
default_edge         = "us"

# Restrict SSH if needed; keep empty for SSM-only.
ssh_allowed_cidrs = []

# Replace with your pushed images (ECR/GHCR/Docker Hub).
edge_image   = "ghcr.io/example/go-cdn:latest"
origin_image = "ghcr.io/example/go-cdn-origin:latest"
dns_image    = "ghcr.io/example/go-cdn-dns:latest"

geo_cidr_rules = [
  "49.36.0.0/14=in",
  "3.0.0.0/8=us",
  "18.0.0.0/8=us"
]
