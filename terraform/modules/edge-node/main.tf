module "impl" {
  source = "../edge"

  project_name      = var.project_name
  region_label      = var.region_label
  instance_type     = var.instance_type
  ssh_allowed_cidrs = var.ssh_allowed_cidrs
  ami_owner         = var.ami_owner
  container_image   = var.container_image
  origin_url        = var.origin_url
}
