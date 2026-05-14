# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build -o /out/auth ./services/auth/cmd

FROM alpine:3.22

RUN apk add --no-cache ca-certificates netcat-openbsd
WORKDIR /app
COPY --from=build /out/auth /app/auth

EXPOSE 8002 8085
CMD ["/app/auth"]
