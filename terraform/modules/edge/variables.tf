variable "project_name" {
  type = string
}

variable "region_label" {
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

variable "origin_url" {
  type = string
}
