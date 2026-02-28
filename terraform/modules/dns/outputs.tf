output "public_ip" {
  value = aws_eip.dns.public_ip
}

output "instance_id" {
  value = aws_instance.dns.id
}
