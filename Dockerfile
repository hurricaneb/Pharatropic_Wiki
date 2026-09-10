# Stage 1: Build React Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm install
COPY web/ ./
RUN npm run build

# Stage 2: Build Go Backend
FROM golang:1.26.6-alpine AS backend-builder
ENV GOTOOLCHAIN=auto
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o wiki-server main.go

# Stage 3: Minimal Runtime Container
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=backend-builder /app/wiki-server ./wiki-server
COPY --from=frontend-builder /app/web/dist ./web/dist

RUN mkdir -p /app/uploads

EXPOSE 8080

ENV PORT=8080
ENV DB_DRIVER=sqlite
ENV DB_PATH=/app/wiki.db

# Reports container status as "healthy"/"unhealthy" (visible in `docker ps`,
# Portainer, and consumable by external monitors like Uptime Kuma) based on
# the app's own /api/v1/health endpoint. wget is busybox-provided on alpine,
# no extra package needed.
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD wget --quiet --tries=1 --spider "http://localhost:${PORT}/api/v1/health" || exit 1

CMD ["/app/wiki-server"]
