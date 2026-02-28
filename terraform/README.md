# Terraform Deployment (AWS Multi-Region)

This setup creates:

- 1 origin EC2 in `us-east-1`
- 3 edge EC2 instances in:
  - `us-east-1`
  - `ap-south-1` (India)
  - `eu-central-1`
- Route53 geo-DNS records for `cdn.<domain>`:
  - country `US` -> US edge
  - country `IN` -> India edge
  - continent `EU` -> EU edge
  - default -> US edge

## DNS Hosting Answer

You do **not** need to run a separate DNS server instance when using Route53.
Route53 is a managed DNS service. You only host app/origin/edge instances.

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
- Existing public Route53 hosted zone (e.g. `example.com.`)
- CDN/origin container images published to ECR/GHCR/Docker Hub

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
