variable "project_name" {
  description = "Project prefix for naming AWS resources."
  type        = string
  default     = "gocdn"
}

variable "domain_name" {
  description = "Route53 hosted zone domain (must already exist), e.g. example.com."
  type        = string
}

variable "cdn_record_name" {
  description = "Record name inside hosted zone used for CDN entrypoint."
  type        = string
  default     = "cdn"
}

variable "origin_record_name" {
  description = "Record name inside hosted zone used for origin host."
  type        = string
  default     = "origin"
}

variable "ssh_allowed_cidrs" {
  description = "Optional CIDRs allowed to access SSH (22). Keep empty for SSM-only access."
  type        = list(string)
  default     = []
}

variable "edge_instance_type" {
  description = "EC2 instance type for edge nodes."
  type        = string
  default     = "t3.small"
}

variable "origin_instance_type" {
  description = "EC2 instance type for origin node."
  type        = string
  default     = "t3.small"
}

variable "edge_ami_owner" {
  description = "AMI owner for Amazon Linux."
  type        = string
  default     = "137112412989"
}

variable "edge_image" {
  description = "Container image to run on edge EC2 instances."
  type        = string
  default     = "ghcr.io/example/go-cdn:latest"
}

variable "origin_image" {
  description = "Container image to run on origin EC2 instance."
  type        = string
  default     = "ghcr.io/example/go-cdn-origin:latest"
}
