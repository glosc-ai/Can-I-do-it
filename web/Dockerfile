# syntax=docker/dockerfile:1.7
#
# Development only. Production builds the SPA as part of the repo-root
# Dockerfile, which embeds the output into the Go binary (see server/webui).

FROM node:22-alpine AS dependencies
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY docker-entrypoint.sh /usr/local/bin/web-entrypoint
RUN chmod +x /usr/local/bin/web-entrypoint

FROM dependencies AS development
COPY . .
ENTRYPOINT ["web-entrypoint"]
EXPOSE 5173
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]
