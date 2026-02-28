project_name         = "gocdn-prod"
authoritative_domain = "cdn.example.com."
edge_instance_type   = "t3.medium"
origin_instance_type = "t3.medium"
dns_instance_type    = "t3.small"
default_edge         = "us"

# Set your office/VPN CIDRs if SSH is required.
ssh_allowed_cidrs = []

# Production image tags should be immutable.
edge_image   = "ghcr.io/example/go-cdn:v1.0.0"
origin_image = "ghcr.io/example/go-cdn-origin:v1.0.0"
dns_image    = "ghcr.io/example/go-cdn-dns:v1.0.0"

geo_cidr_rules = [
  "49.36.0.0/14=in",
  "103.0.0.0/8=in",
  "3.0.0.0/8=us",
  "18.0.0.0/8=us",
  "13.0.0.0/8=eu"
]
