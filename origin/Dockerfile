FROM golang:1.25.7-alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/origin .

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/origin /app/origin
COPY --from=builder /src/public /app/public
EXPOSE 8081
ENTRYPOINT ["/app/origin"]
