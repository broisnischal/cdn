output "edge_us_public_ip" {
  value = module.edge_us.public_ip
}

output "edge_in_public_ip" {
  value = module.edge_in.public_ip
}

output "edge_eu_public_ip" {
  value = module.edge_eu.public_ip
}

output "origin_public_ip" {
  value = module.origin_us.public_ip
}

output "cdn_fqdn" {
  value = module.dns.cdn_record_name
}

output "origin_fqdn" {
  value = module.dns.origin_record_name
}
