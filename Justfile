set shell := ["bash", "-cu"]

default:
  @just --list

fmt:
  @gofmt -w cdn/*.go dns/*.go origin/*.go

test:
  @cd cdn && go test ./...
  @cd dns && go test ./...
  @cd origin && go test ./...

run-origin:
  @cd origin && go run .

run-edge:
  @cd cdn && env $(grep -v '^#' ../.env | xargs) go run .

run-dns:
  @cd dns && env $(grep -v '^#' ../.env | xargs) go run .

docker-build:
  @docker compose -f compose.yaml build

up:
  @docker compose -f compose.yaml up -d --build

down:
  @docker compose -f compose.yaml down -v

logs service="edge":
  @docker compose -f compose.yaml logs -f {{service}}

terraform-init:
  @cd terraform && terraform init

terraform-plan:
  @cd terraform && terraform plan

terraform-apply:
  @cd terraform && terraform apply -auto-approve

terraform-destroy:
  @cd terraform && terraform destroy -auto-approve
