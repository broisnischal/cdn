# Terraform Deployment

This Terraform setup deploys the local CDN stack with Docker provider:

- `gocdn-origin`
- `gocdn-shield`
- `gocdn-edge`
- `gocdn-dns`

## Usage

```bash
cd terraform
terraform init
terraform plan
terraform apply
```

Destroy:

```bash
terraform destroy
```
