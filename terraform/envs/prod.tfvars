project_name         = "gocdn-prod"
authoritative_domain = "jotko.site."
edge_instance_type   = "t3.micro"
origin_instance_type = "t3.micro"
dns_instance_type    = "t3.micro"
default_edge         = "us"
ns_hosts             = ["ns1"]

# Set your office/VPN CIDRs if SSH is required.
ssh_allowed_cidrs = []

# Production image tags should be immutable.
edge_image   = "ghcr.io/broisnischal/go-cdn:latest"
origin_image = "ghcr.io/broisnischal/go-cdn-origin:latest"
dns_image    = "ghcr.io/broisnischal/go-cdn-dns:latest"
# Preferred MaxMind auth settings (per docs):
# download URL: https://download.maxmind.com/geoip/databases/<edition>/download?suffix=tar.gz
dns_geoip_account_id  = "1305731"
dns_geoip_license_key = "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"
dns_geoip_edition_id  = "GeoLite2-City"
dns_geoip_db_url      = "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=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&suffix=tar.gz"


geo_cidr_rules = [
  "49.36.0.0/14=in",
  "103.0.0.0/8=in",
  "3.0.0.0/8=us",
  "18.0.0.0/8=us",
  "13.0.0.0/8=eu"
]
