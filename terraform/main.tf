module "origin_us" {
  source = "./modules/origin-node"
  providers = {
    aws = aws.us
  }

  project_name       = var.project_name
  region_label       = "us-east-1"
  instance_type      = var.origin_instance_type
  ssh_allowed_cidrs  = var.ssh_allowed_cidrs
  ami_owner          = var.edge_ami_owner
  container_image    = var.origin_image
}

module "edge_us" {
  source = "./modules/edge-node"
  providers = {
    aws = aws.us
  }

  project_name      = var.project_name
  region_label      = "us-east-1"
  instance_type     = var.edge_instance_type
  ssh_allowed_cidrs = var.ssh_allowed_cidrs
  ami_owner         = var.edge_ami_owner
  container_image   = var.edge_image
  origin_url        = "http://${module.origin_us.public_ip}:8081"
}

module "edge_in" {
  source = "./modules/edge-node"
  providers = {
    aws = aws.in
  }

  project_name      = var.project_name
  region_label      = "ap-south-1"
  instance_type     = var.edge_instance_type
  ssh_allowed_cidrs = var.ssh_allowed_cidrs
  ami_owner         = var.edge_ami_owner
  container_image   = var.edge_image
  origin_url        = "http://${module.origin_us.public_ip}:8081"
}

module "edge_eu" {
  source = "./modules/edge-node"
  providers = {
    aws = aws.eu
  }

  project_name      = var.project_name
  region_label      = "eu-central-1"
  instance_type     = var.edge_instance_type
  ssh_allowed_cidrs = var.ssh_allowed_cidrs
  ami_owner         = var.edge_ami_owner
  container_image   = var.edge_image
  origin_url        = "http://${module.origin_us.public_ip}:8081"
}

module "dns" {
  source = "./modules/dns"
  providers = {
    aws = aws.us
  }

  project_name           = var.project_name
  instance_type          = var.dns_instance_type
  ssh_allowed_cidrs      = var.ssh_allowed_cidrs
  ami_owner              = var.edge_ami_owner
  container_image        = var.dns_image
  authoritative_domain   = var.authoritative_domain
  default_edge           = var.default_edge
  origin_ip              = module.origin_us.public_ip
  edge_us_ip             = module.edge_us.public_ip
  edge_in_ip             = module.edge_in.public_ip
  edge_eu_ip             = module.edge_eu.public_ip
  edge_us_coords         = var.edge_us_coords
  edge_in_coords         = var.edge_in_coords
  edge_eu_coords         = var.edge_eu_coords
  geo_cidr_rules         = var.geo_cidr_rules
}
