project_name        = "gocdn-dev"
domain_name         = "example.com."
cdn_record_name     = "cdn-dev"
origin_record_name  = "origin-dev"
edge_instance_type  = "t3.small"
origin_instance_type = "t3.small"

# Restrict SSH if needed; keep empty for SSM-only.
ssh_allowed_cidrs = []

# Replace with your pushed images (ECR/GHCR/Docker Hub).
edge_image   = "ghcr.io/example/go-cdn:latest"
origin_image = "ghcr.io/example/go-cdn-origin:latest"
