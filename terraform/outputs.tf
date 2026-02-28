output "edge_url" {
  value = "http://localhost:${var.edge_port}"
}

output "origin_url" {
  value = "http://localhost:${var.origin_port}"
}

output "dns_endpoint" {
  value = "udp://localhost:${var.dns_port}"
}
