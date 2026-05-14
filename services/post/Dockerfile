# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build -o /out/post ./services/post/cmd

FROM alpine:3.22

RUN apk add --no-cache ca-certificates netcat-openbsd
WORKDIR /app
COPY --from=build /out/post /app/post

EXPOSE 8005 8083
CMD ["/app/post"]
