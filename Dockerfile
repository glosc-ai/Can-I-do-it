# syntax=docker/dockerfile:1.7
#
# Single production image: builds the Vue SPA, embeds it into the Go
# binary via server/webui, and ships one distroless container that serves
# both the API and the static frontend. Local dev still uses the separate
# server/Dockerfile + web/Dockerfile (see docker-compose.dev.yml) for Vite HMR.

FROM node:22-alpine AS web-build
WORKDIR /app
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
ARG VITE_API_BASE_URL=/api/v1
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
RUN npm run build

FROM golang:1.25-alpine AS server-build
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ .
COPY --from=web-build /app/dist ./webui/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app .

FROM gcr.io/distroless/static-debian12:nonroot AS production
COPY --from=server-build /out/app /app
ENV HTTP_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/app"]
