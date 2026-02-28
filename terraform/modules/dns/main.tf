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

resource "aws_security_group" "dns" {
  name        = "${var.project_name}-dns"
  description = "Authoritative DNS SG"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    from_port   = 53
    to_port     = 53
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 53
    to_port     = 53
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
  name = "${var.project_name}-dns-ssm"

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
  name = "${var.project_name}-dns-profile"
  role = aws_iam_role.ssm_role.name
}

locals {
  edge_servers = join(",", [
    "us|${var.edge_us_ip}|${var.edge_us_coords.lat}|${var.edge_us_coords.lon}",
    "in|${var.edge_in_ip}|${var.edge_in_coords.lat}|${var.edge_in_coords.lon}",
    "eu|${var.edge_eu_ip}|${var.edge_eu_coords.lat}|${var.edge_eu_coords.lon}"
  ])
}

resource "aws_instance" "dns" {
  ami                         = data.aws_ami.al2023.id
  instance_type               = var.instance_type
  subnet_id                   = data.aws_subnets.default.ids[0]
  vpc_security_group_ids      = [aws_security_group.dns.id]
  iam_instance_profile        = aws_iam_instance_profile.profile.name
  user_data_replace_on_change = true

  metadata_options {
    http_tokens = "required"
  }

  root_block_device {
    encrypted   = true
    volume_size = 20
    volume_type = "gp3"
  }

  user_data = <<-EOF
    #cloud-config
    runcmd:
      - |
          set -euxo pipefail
          retry() {
            local n=0
            local max=5
            local delay=5
            until "$@"; do
              n=$((n+1))
              if [ "$n" -ge "$max" ]; then
                echo "command failed after $n attempts: $*"
                return 1
              fi
              sleep "$delay"
            done
          }

          retry dnf -y update
          retry dnf -y install docker amazon-ssm-agent
          systemctl enable docker
          systemctl start docker
          systemctl enable amazon-ssm-agent
          systemctl restart amazon-ssm-agent

          mkdir -p /opt/geoip
          if [ -n "${var.geoip_account_id}" ] && [ -n "${var.geoip_license_key}" ]; then
            # MaxMind direct download permalink; curl -L follows the R2 redirect host.
            retry curl -fsSL -L \
              -u "${var.geoip_account_id}:${var.geoip_license_key}" \
              "https://download.maxmind.com/geoip/databases/${var.geoip_edition_id}/download?suffix=tar.gz" \
              -o /tmp/geoip-download.bin
          elif [ -n "${var.geoip_db_url}" ]; then
            retry curl -fsSL -L "${var.geoip_db_url}" -o /tmp/geoip-download.bin
          fi
          if [ -s /tmp/geoip-download.bin ]; then
            if tar -tzf /tmp/geoip-download.bin >/dev/null 2>&1; then
              tar -xzf /tmp/geoip-download.bin -C /tmp
              MMDB_SRC=$(ls /tmp/GeoLite2-City_*/GeoLite2-City.mmdb 2>/dev/null | head -n1 || true)
              if [ -n "$MMDB_SRC" ]; then
                cp "$MMDB_SRC" /opt/geoip/GeoLite2-City.mmdb
              fi
            else
              cp /tmp/geoip-download.bin /opt/geoip/GeoLite2-City.mmdb
            fi
            chmod 0644 /opt/geoip/GeoLite2-City.mmdb || true
          fi

          TOKEN=$(curl -X PUT "http://169.254.169.254/latest/api/token" -H "X-aws-ec2-metadata-token-ttl-seconds: 21600")
          PUBLIC_IP=$(curl -H "X-aws-ec2-metadata-token: $TOKEN" -s http://169.254.169.254/latest/meta-data/public-ipv4)

          docker rm -f authoritative-dns || true
          retry docker pull ${var.container_image}
          docker run -d --name authoritative-dns --restart unless-stopped \
            --cap-add NET_BIND_SERVICE \
            --user 0:0 \
            --network host \
            -v /opt/geoip:/data/geoip:ro \
            -e DNS_LISTEN_ADDR=:53 \
            -e DNS_SELF_IP=$PUBLIC_IP \
            -e DNS_NS_HOSTS='${join(",", var.ns_hosts)}' \
            -e DNS_GEOIP_DB_PATH=/data/geoip/GeoLite2-City.mmdb \
            -e DNS_AUTHORITATIVE_DOMAIN=${var.authoritative_domain} \
            -e DNS_ORIGIN_IP=${var.origin_ip} \
            -e DNS_DEFAULT_EDGE=${var.default_edge} \
            -e DNS_EDGE_SERVERS='${local.edge_servers}' \
            -e DNS_GEO_CIDR_RULES='${join(",", var.geo_cidr_rules)}' \
            ${var.container_image}
  EOF

  tags = {
    Name = "${var.project_name}-dns-authoritative"
    Role = "dns"
  }
}

resource "aws_eip" "dns" {
  domain   = "vpc"
  instance = aws_instance.dns.id

  tags = {
    Name = "${var.project_name}-dns-eip"
  }
}
