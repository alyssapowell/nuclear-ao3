# One image with the whole archive: the website and the backend in a single
# program, for hosting platforms that run one container and put HTTPS in
# front of it themselves (see "On a hosting platform" in docs/HOSTING.md).
# The docker-compose.yml setup doesn't use this file; it builds
# backend/Dockerfile and frontend/Dockerfile and puts Caddy in front.

# The website, built into plain files. Node is only used here, to build.
FROM node:22.23.3-alpine AS website
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

# The backend.
FROM golang:1.23.4-alpine3.21 AS backend
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /nuclear ./cmd/nuclear

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 1001 app
COPY --from=backend /nuclear /usr/local/bin/nuclear
COPY --from=website /app/dist /srv
# The app serves the website from /srv as well as the API, so nothing else
# is needed in front of it but HTTPS.
ENV STATIC_DIR=/srv PORT=8080
USER app
EXPOSE 8080
CMD ["nuclear"]
