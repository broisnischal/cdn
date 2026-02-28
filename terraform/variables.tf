variable "project_name" {
  description = "Project prefix for naming AWS resources."
  type        = string
  default     = "gocdn"
}

variable "authoritative_domain" {
  description = "Domain served by your authoritative DNS server, e.g. cdn.example.com."
  type        = string
  default     = "jotko.site."
}

variable "ssh_allowed_cidrs" {
  description = "Optional CIDRs allowed to access SSH (22). Keep empty for SSM-only access."
  type        = list(string)
  default     = []
}

variable "edge_instance_type" {
  description = "EC2 instance type for edge nodes."
  type        = string
  default     = "t3.micro"
}

variable "origin_instance_type" {
  description = "EC2 instance type for origin node."
  type        = string
  default     = "t3.micro"
}

variable "dns_instance_type" {
  description = "EC2 instance type for authoritative DNS node."
  type        = string
  default     = "t3.micro"
}

variable "edge_ami_owner" {
  description = "AMI owner for Amazon Linux."
  type        = string
  default     = "137112412989"
}

variable "edge_image" {
  description = "Container image to run on edge EC2 instances."
  type        = string
  default     = "ghcr.io/broisnischal/go-cdn:latest"
}

variable "origin_image" {
  description = "Container image to run on origin EC2 instance."
  type        = string
  default     = "ghcr.io/broisnischal/go-cdn-origin:latest"
}

variable "dns_image" {
  description = "Container image to run on DNS EC2 instance."
  type        = string
  default     = "ghcr.io/broisnischal/go-cdn-dns:latest"
}

variable "default_edge" {
  description = "Default edge name when no CIDR/GeoIP match is found."
  type        = string
  default     = "us"
}

variable "edge_us_coords" {
  description = "Latitude/longitude for US edge."
  type        = object({ lat = number, lon = number })
  default     = { lat = 37.7749, lon = -122.4194 }
}

variable "edge_in_coords" {
  description = "Latitude/longitude for India edge."
  type        = object({ lat = number, lon = number })
  default     = { lat = 19.0760, lon = 72.8777 }
}

variable "edge_eu_coords" {
  description = "Latitude/longitude for EU edge."
  type        = object({ lat = number, lon = number })
  default     = { lat = 50.1109, lon = 8.6821 }
}

variable "geo_cidr_rules" {
  description = "CIDR to edge routing rules in format CIDR=edge, e.g. 49.36.0.0/14=in."
  type        = list(string)
  default     = [
    "49.36.0.0/14=in",
    "8.8.8.0/24=us"
  ]
}

variable "ns_hosts" {
  description = "Authoritative NS host labels served under authoritative_domain."
  type        = list(string)
  default     = ["ns1"]
}
