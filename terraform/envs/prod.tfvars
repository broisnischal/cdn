project_name         = "gocdn-prod"
domain_name          = "example.com."
cdn_record_name      = "cdn"
origin_record_name   = "origin"
edge_instance_type   = "t3.medium"
origin_instance_type = "t3.medium"

# Set your office/VPN CIDRs if SSH is required.
ssh_allowed_cidrs = []

# Production image tags should be immutable.
edge_image   = "ghcr.io/example/go-cdn:v1.0.0"
origin_image = "ghcr.io/example/go-cdn-origin:v1.0.0"
