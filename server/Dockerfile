# syntax=docker/dockerfile:1.7
#
# Development only. Production builds use the repo-root Dockerfile, which
# embeds the built Vue SPA into the Go binary (see server/webui).

FROM golang:1.25-alpine AS development
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download
COPY . .
EXPOSE 8080
CMD ["go", "run", "."]
