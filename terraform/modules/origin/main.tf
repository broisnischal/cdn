data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

data "aws_ami" "al2023" {
  most_recent = true
  owners      = [var.ami_owner]

  filter {
    name   = "name"
    values = ["al2023-ami-*-x86_64"]
  }
}

resource "aws_security_group" "origin" {
  name        = "${var.project_name}-origin-${var.region_label}"
  description = "Origin node SG"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    from_port   = 8081
    to_port     = 8081
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  dynamic "ingress" {
    for_each = var.ssh_allowed_cidrs
    content {
      from_port   = 22
      to_port     = 22
      protocol    = "tcp"
      cidr_blocks = [ingress.value]
    }
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_iam_role" "ssm_role" {
  name = "${var.project_name}-origin-ssm-${var.region_label}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ec2.amazonaws.com"
      }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "ssm" {
  role       = aws_iam_role.ssm_role.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_instance_profile" "profile" {
  name = "${var.project_name}-origin-profile-${var.region_label}"
  role = aws_iam_role.ssm_role.name
}

resource "aws_instance" "origin" {
  ami                    = data.aws_ami.al2023.id
  instance_type          = var.instance_type
  subnet_id              = data.aws_subnets.default.ids[0]
  vpc_security_group_ids = [aws_security_group.origin.id]
  iam_instance_profile   = aws_iam_instance_profile.profile.name

  metadata_options {
    http_tokens = "required"
  }

  root_block_device {
    encrypted = true
    volume_size = 20
    volume_type = "gp3"
  }

  user_data = <<-EOF
    #!/bin/bash
    set -euxo pipefail
    dnf -y update
    dnf -y install docker
    systemctl enable docker
    systemctl start docker
    docker run -d --name origin --restart unless-stopped \
      -p 8081:8081 \
      ${var.container_image}
  EOF

  tags = {
    Name      = "${var.project_name}-origin-${var.region_label}"
    Role      = "origin"
    RegionTag = var.region_label
  }
}

resource "aws_eip" "origin" {
  domain   = "vpc"
  instance = aws_instance.origin.id

  tags = {
    Name = "${var.project_name}-origin-eip-${var.region_label}"
  }
}
