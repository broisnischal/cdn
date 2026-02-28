variable "project_name" {
  type = string
}

variable "instance_type" {
  type = string
}

variable "ssh_allowed_cidrs" {
  type = list(string)
}

variable "ami_owner" {
  type = string
}

variable "container_image" {
  type = string
}

variable "authoritative_domain" {
  type = string
}

variable "default_edge" {
  type = string
}

variable "edge_us_ip" {
  type = string
}

variable "edge_in_ip" {
  type = string
}

variable "edge_eu_ip" {
  type = string
}

variable "origin_ip" {
  type = string
}

variable "edge_us_coords" {
  type = object({ lat = number, lon = number })
}

variable "edge_in_coords" {
  type = object({ lat = number, lon = number })
}

variable "edge_eu_coords" {
  type = object({ lat = number, lon = number })
}

variable "geo_cidr_rules" {
  type = list(string)
}

variable "ns_hosts" {
  type = list(string)
}
