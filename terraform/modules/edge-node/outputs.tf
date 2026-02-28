output "public_ip" {
  value = aws_eip.edge.public_ip
}

output "instance_id" {
  value = aws_instance.edge.id
}
