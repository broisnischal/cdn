locals {
  project_root = "${path.module}/.."
}

resource "docker_network" "cdnnet" {
  name = "gocdn-net"
  ipam_config {
    subnet = "172.29.0.0/16"
  }
}

resource "docker_volume" "edge_cache" {
  name = "gocdn-edge-cache"
}

resource "docker_volume" "shield_cache" {
  name = "gocdn-shield-cache"
}

resource "docker_image" "origin" {
  name = "gocdn-origin:latest"
  build {
    context    = "${local.project_root}/origin"
    dockerfile = "Dockerfile"
  }
}

resource "docker_image" "cdn" {
  name = "gocdn-cdn:latest"
  build {
    context    = "${local.project_root}/cdn"
    dockerfile = "Dockerfile"
  }
}

resource "docker_image" "dns" {
  name = "gocdn-dns:latest"
  build {
    context    = "${local.project_root}/dns"
    dockerfile = "Dockerfile"
  }
}

resource "docker_container" "origin" {
  name  = "gocdn-origin"
  image = docker_image.origin.image_id

  ports {
    internal = 8081
    external = var.origin_port
  }

  networks_advanced {
    name         = docker_network.cdnnet.name
    ipv4_address = "172.29.0.10"
  }
}

resource "docker_container" "shield" {
  name  = "gocdn-shield"
  image = docker_image.cdn.image_id

  env = [
    "EDGE_LISTEN_ADDR=:8080",
    "ORIGIN_URL=http://gocdn-origin:8081",
    "EDGE_MAX_MEMORY_BYTES=134217728",
    "EDGE_EVICTION_POLICY=lru",
    "EDGE_DISK_CACHE_DIR=/cache",
    "EDGE_DISK_CACHE_MAX_BYTES=2147483648",
    "UPSTREAM_TIMEOUT_SEC=10",
  ]

  volumes {
    volume_name    = docker_volume.shield_cache.name
    container_path = "/cache"
  }

  depends_on = [docker_container.origin]

  networks_advanced {
    name         = docker_network.cdnnet.name
    ipv4_address = "172.29.0.15"
  }
}

resource "docker_container" "edge" {
  name  = "gocdn-edge"
  image = docker_image.cdn.image_id

  env = [
    "EDGE_LISTEN_ADDR=:8080",
    "SHIELD_URL=http://gocdn-shield:8080",
    "EDGE_MAX_MEMORY_BYTES=268435456",
    "EDGE_EVICTION_POLICY=lru",
    "EDGE_DISK_CACHE_DIR=/cache",
    "EDGE_DISK_CACHE_MAX_BYTES=4294967296",
    "UPSTREAM_TIMEOUT_SEC=10",
  ]

  ports {
    internal = 8080
    external = var.edge_port
  }

  volumes {
    volume_name    = docker_volume.edge_cache.name
    container_path = "/cache"
  }

  depends_on = [docker_container.shield]

  networks_advanced {
    name         = docker_network.cdnnet.name
    ipv4_address = "172.29.0.20"
  }
}

resource "docker_container" "dns" {
  name  = "gocdn-dns"
  image = docker_image.dns.image_id

  env = [
    "DNS_LISTEN_ADDR=:5353",
    "DNS_DOMAIN=cdn.local.",
    "DNS_DEFAULT_POOL=default",
    "DNS_POOL_DEFAULT=172.29.0.20:100",
    "DNS_GEO_RULES=10.0.0.0/8=default,192.168.0.0/16=default",
  ]

  ports {
    internal = 5353
    external = var.dns_port
    protocol = "udp"
  }

  depends_on = [docker_container.edge]

  networks_advanced {
    name         = docker_network.cdnnet.name
    ipv4_address = "172.29.0.53"
  }
}
