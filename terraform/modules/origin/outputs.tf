output "public_ip" {
  value = aws_eip.origin.public_ip
}

output "instance_id" {
  value = aws_instance.origin.id
}
