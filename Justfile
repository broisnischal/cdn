registry := "ghcr.io"
owner := "broisnischal"
tag := "latest"

default:
  just --list

fmt:
  gofmt -w cdn/*.go dns/*.go origin/*.go

test:
  cd cdn && go test ./...
  cd dns && go test ./...
  cd origin && go test ./...

run-origin:
  cd origin && go run . 

run-edge:
  cd cdn && env $(grep -v '^#' ../.env | xargs) go run .

run-dns:
  cd dns && env $(grep -v '^#' ../.env | xargs) go run .

run-dns-53:
  cd dns && sudo env $(grep -v '^#' ../.env | xargs) DNS_LISTEN_ADDR=:53 go run .

docker-build:
  docker compose -f compose.yaml build

docker-build-images tag=tag:
  docker build -t {{registry}}/{{owner}}/go-cdn:{{tag}} ./cdn
  docker build -t {{registry}}/{{owner}}/go-cdn-origin:{{tag}} ./origin
  docker build -t {{registry}}/{{owner}}/go-cdn-dns:{{tag}} ./dns

ghcr-login:
  test -n "${GHCR_TOKEN:-}" || (echo "GHCR_TOKEN is required" && exit 1)
  echo "${GHCR_TOKEN}" | docker login {{registry}} -u {{owner}} --password-stdin

docker-push-images tag=tag:
  docker push {{registry}}/{{owner}}/go-cdn:{{tag}}
  docker push {{registry}}/{{owner}}/go-cdn-origin:{{tag}}
  docker push {{registry}}/{{owner}}/go-cdn-dns:{{tag}}

publish-images tag=tag:
  just ghcr-login
  just docker-build-images {{tag}}
  just docker-push-images {{tag}}

up:
  docker compose -f compose.yaml up -d --build

down:
  docker compose -f compose.yaml down -v

logs service="edge":
  docker compose -f compose.yaml logs -f {{service}}

terraform-init:
  cd terraform && terraform init

terraform-plan:
  cd terraform && terraform plan

terraform-apply:
  cd terraform && terraform apply -auto-approve

terraform-destroy:
  cd terraform && terraform destroy -auto-approve
