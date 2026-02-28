# Terraform Deployment (AWS Multi-Region, Own Authoritative DNS)

This setup creates:

- 1 origin EC2 in `us-east-1`
- 3 edge EC2 instances in:
  - `us-east-1`
  - `ap-south-1` (India)
  - `eu-central-1`
- 1 authoritative DNS EC2 in `us-east-1` running your DNS service on `:53` UDP/TCP

DNS routing logic is inside your DNS service:

- CIDR match (`DNS_GEO_CIDR_RULES`) first
- then GeoIP lookup + Haversine nearest-edge
- fallback to `DNS_DEFAULT_EDGE`

## Folder Layout

```text
terraform/
  providers.tf
  main.tf
  variables.tf
  outputs.tf
  modules/
    edge-node/
    origin-node/
    dns/
  envs/
    dev.tfvars
    prod.tfvars
```

## Prerequisites

- Terraform >= 1.6
- AWS credentials configured (`aws configure`, env vars, or SSO)
- CDN/origin container images published to ECR/GHCR/Docker Hub
- DNS image published to ECR/GHCR/Docker Hub

## Deploy (dev)

```bash
cd terraform
terraform init
terraform plan -var-file=envs/dev.tfvars
terraform apply -var-file=envs/dev.tfvars
```

## Deploy (prod)

```bash
terraform plan -var-file=envs/prod.tfvars
terraform apply -var-file=envs/prod.tfvars
```

## Destroy

```bash
terraform destroy -var-file=envs/dev.tfvars
```

## Production Notes

- SSH is disabled by default (`ssh_allowed_cidrs = []`), use SSM Session Manager.
- IMDSv2 is enforced on instances.
- Root volumes are encrypted.
- Prefer immutable image tags in production.
- To make this truly authoritative globally, register your DNS node(s) as NS at your registrar and configure glue records (`ns1/ns2`).
