resource "aws_route53_record" "cdn_us" {
  zone_id = var.zone_id
  name    = var.cdn_record_name
  type    = "A"
  ttl     = 30
  records = [var.edge_us_ip]

  set_identifier = "us"
  geolocation_routing_policy {
    country = "US"
  }
}

resource "aws_route53_record" "cdn_in" {
  zone_id = var.zone_id
  name    = var.cdn_record_name
  type    = "A"
  ttl     = 30
  records = [var.edge_in_ip]

  set_identifier = "india"
  geolocation_routing_policy {
    country = "IN"
  }
}

resource "aws_route53_record" "cdn_eu" {
  zone_id = var.zone_id
  name    = var.cdn_record_name
  type    = "A"
  ttl     = 30
  records = [var.edge_eu_ip]

  set_identifier = "eu-default"
  geolocation_routing_policy {
    continent = "EU"
  }
}

resource "aws_route53_record" "cdn_default" {
  zone_id = var.zone_id
  name    = var.cdn_record_name
  type    = "A"
  ttl     = 30
  records = [var.edge_us_ip]

  set_identifier = "default"
  geolocation_routing_policy {
    country = "*"
  }
}

resource "aws_route53_record" "origin" {
  zone_id = var.zone_id
  name    = var.origin_record_name
  type    = "A"
  ttl     = 30
  records = [var.origin_ip]
}
