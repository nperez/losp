FROM golang:1.24-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go generate ./internal/stdlib/ && CGO_ENABLED=0 go build -o /losp ./cmd/losp

FROM debian:bookworm-slim
COPY --from=build /losp /usr/local/bin/losp
WORKDIR /app
ENTRYPOINT ["losp"]
