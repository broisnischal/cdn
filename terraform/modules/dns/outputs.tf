output "cdn_record_name" {
  value = aws_route53_record.cdn_default.fqdn
}

output "origin_record_name" {
  value = aws_route53_record.origin.fqdn
}
